package agentgateway

import (
	"bytes"
	"context"
	"fmt"
	"github.com/swarm-deploy/webroute/api"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/buildkite/interpolate"
	"gopkg.in/yaml.v3"
)

// AgentgatewayProvider resolves routes configured in agentgateway YAML configs.
type AgentgatewayProvider struct{}

func NewAgentgatewayProvider() *AgentgatewayProvider {
	return &AgentgatewayProvider{}
}

// Resolve resolves agentgateway HTTP routes from mounted YAML configs.
func (*AgentgatewayProvider) Resolve(ctx context.Context, service api.Service) ([]api.WebRoute, error) {
	var routes []api.WebRoute
	var env map[string]string
	envLoaded := false

	for _, config := range service.Configs() {
		if !isAgentgatewayYAMLConfig(config.Path()) {
			continue
		}

		if !envLoaded {
			var err error
			env, err = service.Environment()
			if err != nil {
				return nil, fmt.Errorf("get environment variables: %w", err)
			}
			envLoaded = true
		}

		configRoutes, err := resolveAgentgatewayConfigRoutes(ctx, config, env)
		if err != nil {
			return nil, err
		}

		routes = append(routes, configRoutes...)
	}

	return routes, nil
}

func resolveAgentgatewayConfigRoutes(
	ctx context.Context,
	config api.ServiceConfig,
	env map[string]string,
) ([]api.WebRoute, error) {
	var b bytes.Buffer
	if err := config.Read(ctx, &b); err != nil {
		return nil, fmt.Errorf("read agentgateway config %q: %w", config.Path(), err)
	}

	var parsed struct {
		Gateways map[string]agentgatewayConfigGateway `yaml:"gateways"`
		Routes   []agentgatewayConfigRoute            `yaml:"routes"`
	}

	if err := yaml.Unmarshal(b.Bytes(), &parsed); err != nil {
		return nil, fmt.Errorf("parse agentgateway config %q: %w", config.Path(), err)
	}

	interpolateEnv := interpolate.NewMapEnv(env)
	defaultGatewayPort := strings.TrimSpace(parsed.Gateways["default"].Port)
	gatewayPort, interpolateErr := interpolateAgentgatewayValue(defaultGatewayPort, interpolateEnv)
	if interpolateErr != nil {
		return nil, fmt.Errorf("interpolate agentgateway default gateway port in %q: %w", config.Path(), interpolateErr)
	}

	return resolveAgentgatewayRoutes(config.Path(), parsed.Routes, gatewayPort, interpolateEnv)
}

func resolveAgentgatewayRoutes(
	configPath string,
	configRoutes []agentgatewayConfigRoute,
	gatewayPort string,
	env interpolate.Env,
) ([]api.WebRoute, error) {
	var routes []api.WebRoute
	for _, configRoute := range configRoutes {
		if !isAgentgatewayRoute(configRoute) {
			continue
		}

		configRouteValues, err := resolveAgentgatewayRoute(configPath, configRoute, gatewayPort, env)
		if err != nil {
			return nil, err
		}

		routes = append(routes, configRouteValues...)
	}

	return routes, nil
}

func resolveAgentgatewayRoute(
	configPath string,
	configRoute agentgatewayConfigRoute,
	gatewayPort string,
	env interpolate.Env,
) ([]api.WebRoute, error) {
	fromValues, err := agentgatewayFromValues(configRoute, env)
	if err != nil {
		return nil, fmt.Errorf("interpolate agentgateway route match in %q: %w", configPath, err)
	}

	toValues, err := agentgatewayToValues(configRoute.Backends, env)
	if err != nil {
		return nil, fmt.Errorf("interpolate agentgateway route backend in %q: %w", configPath, err)
	}

	var routes []api.WebRoute
	for _, fromValue := range fromValues {
		from, parseErr := agentgatewayFromAddress(fromValue.Hostname, fromValue.Path, gatewayPort)
		if parseErr != nil {
			return nil, fmt.Errorf("parse agentgateway route hostname %q in %q: %w", fromValue.Hostname, configPath, parseErr)
		}

		fromRoutes, routeErr := agentgatewayRoutesForFrom(configPath, from, toValues)
		if routeErr != nil {
			return nil, routeErr
		}

		routes = append(routes, fromRoutes...)
	}

	return routes, nil
}

func agentgatewayRoutesForFrom(configPath string, from api.Address, toValues []string) ([]api.WebRoute, error) {
	if len(toValues) == 0 {
		return []api.WebRoute{{
			Provider: api.ProviderNameAgentgateway,
			From:     from,
		}}, nil
	}

	routes := make([]api.WebRoute, 0, len(toValues))
	for _, toValue := range toValues {
		to, err := agentgatewayBackendAddress(toValue)
		if err != nil {
			return nil, fmt.Errorf("parse agentgateway route backend %q in %q: %w", toValue, configPath, err)
		}

		routes = append(routes, api.WebRoute{
			Provider: api.ProviderNameAgentgateway,
			From:     from,
			To:       &to,
		})
	}

	return routes, nil
}

type agentgatewayConfigGateway struct {
	Port string `yaml:"port"`
}

type agentgatewayConfigRoute struct {
	Hostnames []string                  `yaml:"hostnames"`
	Matches   []agentgatewayConfigMatch `yaml:"matches"`
	Backends  []agentgatewayBackend     `yaml:"backends"`
}

type agentgatewayConfigMatch struct {
	Path agentgatewayPathMatcher `yaml:"path"`
}

type agentgatewayPathMatcher struct {
	Exact      string `yaml:"exact"`
	PathPrefix string `yaml:"pathPrefix"`
	Regex      string `yaml:"regex"`
}

type agentgatewayBackend struct {
	Host string                 `yaml:"host"`
	MCP  agentgatewayMCPBackend `yaml:"mcp"`
}

type agentgatewayMCPBackend struct {
	Targets []agentgatewayMCPTarget `yaml:"targets"`
}

type agentgatewayMCPTarget struct {
	MCP agentgatewayMCPHost `yaml:"mcp"`
}

type agentgatewayMCPHost struct {
	Host string `yaml:"host"`
}

type agentgatewayFromValue struct {
	Hostname string
	Path     string
}

func isAgentgatewayYAMLConfig(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func isAgentgatewayRoute(route agentgatewayConfigRoute) bool {
	return len(route.Hostnames) > 0 || len(route.Matches) > 0 || len(route.Backends) > 0
}

func agentgatewayFromValues(route agentgatewayConfigRoute, env interpolate.Env) ([]agentgatewayFromValue, error) {
	hostnames := route.Hostnames
	if len(hostnames) == 0 {
		hostnames = []string{""}
	}

	paths, err := agentgatewayRoutePaths(route.Matches, env)
	if err != nil {
		return nil, err
	}

	values := make([]agentgatewayFromValue, 0, len(hostnames)*len(paths))
	for _, hostname := range hostnames {
		hostname, err = interpolateAgentgatewayValue(hostname, env)
		if err != nil {
			return nil, err
		}

		for _, path := range paths {
			values = append(values, agentgatewayFromValue{
				Hostname: hostname,
				Path:     path,
			})
		}
	}

	return values, nil
}

func agentgatewayRoutePaths(matches []agentgatewayConfigMatch, env interpolate.Env) ([]string, error) {
	if len(matches) == 0 {
		return []string{"/"}, nil
	}

	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		path := strings.TrimSpace(match.Path.Exact)
		if path == "" {
			path = strings.TrimSpace(match.Path.PathPrefix)
		}
		if path == "" {
			path = strings.TrimSpace(match.Path.Regex)
		}
		if path == "" {
			path = "/"
		}

		path, err := interpolateAgentgatewayValue(path, env)
		if err != nil {
			return nil, err
		}
		paths = append(paths, normalizeAgentgatewayPath(path))
	}

	return paths, nil
}

func agentgatewayToValues(backends []agentgatewayBackend, env interpolate.Env) ([]string, error) {
	values := make([]string, 0, len(backends))
	for _, backend := range backends {
		host, err := interpolateAgentgatewayValue(strings.TrimSpace(backend.Host), env)
		if err != nil {
			return nil, err
		}
		if host != "" {
			values = append(values, host)
		}

		for _, target := range backend.MCP.Targets {
			host, err = interpolateAgentgatewayValue(strings.TrimSpace(target.MCP.Host), env)
			if err != nil {
				return nil, err
			}
			if host == "" {
				continue
			}

			values = append(values, host)
		}
	}

	return values, nil
}

func interpolateAgentgatewayValue(value string, env interpolate.Env) (string, error) {
	return interpolate.Interpolate(env, value)
}

func agentgatewayFromAddress(hostname, path, defaultPort string) (api.Address, error) {
	hostname = strings.TrimSpace(hostname)
	path = normalizeAgentgatewayPath(path)
	defaultPort = strings.TrimSpace(defaultPort)
	if hostname == "" {
		return api.Address{
			Address: path,
			Port:    defaultPort,
		}, nil
	}

	parsed, err := url.Parse("//" + hostname)
	if err != nil {
		return api.Address{}, err
	}
	if parsed.Host == "" {
		return api.Address{}, fmt.Errorf("expected hostname")
	}

	port := parsed.Port()
	addressHost := parsed.Host
	if port == "" && defaultPort != "" {
		port = defaultPort
		addressHost += ":" + defaultPort
	}

	return api.Address{
		Address: addressHost + path,
		Domain:  parsed.Hostname(),
		Port:    port,
	}, nil
}

func agentgatewayBackendAddress(value string) (api.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return api.Address{}, fmt.Errorf("expected backend host")
	}

	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return api.Address{}, err
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return api.Address{}, fmt.Errorf("expected absolute URL with scheme and host")
		}

		return api.Address{
			Address: agentgatewayURLAddress(parsed),
			Domain:  parsed.Hostname(),
			Port:    parsed.Port(),
		}, nil
	}

	parsed, err := url.Parse("//" + value)
	if err != nil {
		return api.Address{}, err
	}
	if parsed.Host == "" {
		return api.Address{}, fmt.Errorf("expected backend host")
	}

	return api.Address{
		Address: parsed.Host + parsed.EscapedPath(),
		Domain:  parsed.Hostname(),
		Port:    parsed.Port(),
	}, nil
}

func agentgatewayURLAddress(parsed *url.URL) string {
	address := parsed.Host
	if parsed.EscapedPath() != "" {
		address += parsed.EscapedPath()
	}
	if parsed.RawQuery != "" {
		address += "?" + parsed.RawQuery
	}

	return address
}

func normalizeAgentgatewayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}

	return "/" + path
}
