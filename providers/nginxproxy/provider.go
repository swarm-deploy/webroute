package nginxproxy

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/swarm-deploy/webroute/api"
	"gopkg.in/yaml.v3"
)

const (
	nginxVirtualHostKey           = "VIRTUAL_HOST"
	nginxVirtualHostMultiportsKey = "VIRTUAL_HOST_MULTIPORTS"
	nginxVirtualPathKey           = "VIRTUAL_PATH"
	nginxVirtualPortKey           = "VIRTUAL_PORT"
)

// NginxProxyProvider resolves routes configured for nginx-proxy.
type NginxProxyProvider struct{}

func NewNginxProxyProvider() *NginxProxyProvider {
	return &NginxProxyProvider{}
}

// Resolve resolves nginx-proxy routes from env values.
func (*NginxProxyProvider) Resolve(_ context.Context, service api.Service) ([]api.WebRoute, error) {
	env, err := service.Environment()
	if err != nil {
		return nil, fmt.Errorf("get environment variables: %w", err)
	}

	if len(env) == 0 {
		return nil, nil
	}

	virtualHostMultiports := strings.TrimSpace(env[nginxVirtualHostMultiportsKey])
	if virtualHostMultiports != "" {
		routes, rerr := resolveNginxMultiportRoutes(virtualHostMultiports)
		if rerr != nil {
			return nil, rerr
		}

		return routes, nil
	}

	virtualHosts := strings.TrimSpace(env[nginxVirtualHostKey])
	if virtualHosts == "" {
		return nil, nil
	}

	virtualPath := normalizeNginxPath(env[nginxVirtualPathKey])
	virtualPort := strings.TrimSpace(env[nginxVirtualPortKey])

	routes := make([]api.WebRoute, 0)
	for _, host := range strings.Split(virtualHosts, ",") {
		domain := strings.TrimSpace(host)
		if domain == "" {
			continue
		}
		routeDomain, routeHost := parseNginxHost(domain)

		routes = append(routes, api.WebRoute{
			Provider: api.ProviderNameNginxProxy,
			From: api.Address{
				Domain:  routeDomain,
				Address: nginxRouteAddress(routeHost, virtualPath),
				Port:    virtualPort,
			},
		})
	}

	return routes, nil
}

func resolveNginxMultiportRoutes(raw string) ([]api.WebRoute, error) {
	var hosts map[string]yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &hosts); err != nil {
		return nil, fmt.Errorf("parse %s: %w", nginxVirtualHostMultiportsKey, err)
	}

	hostNames := make([]string, 0, len(hosts))
	for host := range hosts {
		hostNames = append(hostNames, host)
	}
	sort.Strings(hostNames)

	routes := make([]api.WebRoute, 0)
	for _, host := range hostNames {
		hostDomain, routeHost := parseNginxHost(host)
		if hostDomain == "" {
			continue
		}

		hostRoutes := resolveNginxMultiportHostRoutes(routeHost, hostDomain, hosts[host])
		routes = append(routes, hostRoutes...)
	}

	return routes, nil
}

func resolveNginxMultiportHostRoutes(routeHost string, hostDomain string, hostConfig yaml.Node) []api.WebRoute {
	if hostConfig.Kind != yaml.MappingNode {
		return []api.WebRoute{newNginxRoute(routeHost, hostDomain, "/", "")}
	}

	type pathRoute struct {
		path string
		port string
	}

	pathRoutes := make([]pathRoute, 0)
	for idx := 0; idx < len(hostConfig.Content)-1; idx += 2 {
		key := strings.TrimSpace(hostConfig.Content[idx].Value)
		if key == "" || isNginxMultiportHostOption(key) {
			continue
		}

		pathRoutes = append(pathRoutes, pathRoute{
			path: key,
			port: nginxMultiportPathPort(hostConfig.Content[idx+1]),
		})
	}
	sort.Slice(pathRoutes, func(i, j int) bool {
		return pathRoutes[i].path < pathRoutes[j].path
	})

	if len(pathRoutes) == 0 {
		return []api.WebRoute{newNginxRoute(routeHost, hostDomain, "/", "")}
	}

	routes := make([]api.WebRoute, 0, len(pathRoutes))
	for _, pathRoute := range pathRoutes {
		routes = append(routes, newNginxRoute(routeHost, hostDomain, pathRoute.path, pathRoute.port))
	}

	return routes
}

func isNginxMultiportHostOption(key string) bool {
	switch key {
	case "external_http_port", "external_https_port":
		return true
	default:
		return false
	}
}

func nginxMultiportPathPort(pathConfig *yaml.Node) string {
	if pathConfig == nil || pathConfig.Kind != yaml.MappingNode {
		return ""
	}

	for idx := 0; idx < len(pathConfig.Content)-1; idx += 2 {
		key := strings.TrimSpace(pathConfig.Content[idx].Value)
		if key != "port" {
			continue
		}

		return strings.TrimSpace(pathConfig.Content[idx+1].Value)
	}

	return ""
}

func newNginxRoute(routeHost string, hostDomain string, path string, port string) api.WebRoute {
	return api.WebRoute{
		Provider: api.ProviderNameNginxProxy,
		From: api.Address{
			Domain:  hostDomain,
			Address: nginxRouteAddress(routeHost, path),
			Port:    strings.TrimSpace(port),
		},
	}
}

func parseNginxHost(host string) (domain string, addressHost string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", ""
	}
	if strings.HasPrefix(host, "~") {
		return host, host
	}

	parsed, err := url.Parse("//" + host)
	if err != nil || parsed.Host == "" {
		return host, host
	}

	domain = parsed.Hostname()
	if domain == "" {
		domain = host
	}

	return domain, parsed.Host
}

func nginxRouteAddress(host string, path string) string {
	path = normalizeNginxPath(path)
	if strings.HasPrefix(path, "/") {
		return host + path
	}

	return host + "/" + path
}

func normalizeNginxPath(path string) string {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return "/"
	}
	if strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "~") {
		return normalized
	}

	return "/" + normalized
}
