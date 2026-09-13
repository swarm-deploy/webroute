package webroute

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PomeriumProvider resolves routes configured in Pomerium YAML configs.
type PomeriumProvider struct{}

func NewPomeriumProvider() *PomeriumProvider {
	return &PomeriumProvider{}
}

// Resolve resolves Pomerium routes from mounted YAML configs.
func (*PomeriumProvider) Resolve(ctx context.Context, service Service) ([]Route, error) {
	var routes []Route

	for _, config := range service.Configs() {
		if !isPomeriumYAMLConfig(config.Path()) {
			continue
		}

		configRoutes, err := resolvePomeriumConfigRoutes(ctx, config)
		if err != nil {
			return nil, err
		}

		routes = append(routes, configRoutes...)
	}

	return routes, nil
}

func resolvePomeriumConfigRoutes(ctx context.Context, config ServiceConfig) ([]Route, error) {
	var b bytes.Buffer
	if err := config.Read(ctx, &b); err != nil {
		return nil, fmt.Errorf("read pomerium config %q: %w", config.Path(), err)
	}

	var parsed struct {
		Routes []struct {
			From string    `yaml:"from"`
			To   yaml.Node `yaml:"to"`
		} `yaml:"routes"`
	}

	if err := yaml.Unmarshal(b.Bytes(), &parsed); err != nil {
		return nil, fmt.Errorf("parse pomerium config %q: %w", config.Path(), err)
	}

	routes := make([]Route, 0, len(parsed.Routes))
	for _, configRoute := range parsed.Routes {
		fromValue := strings.TrimSpace(configRoute.From)
		if fromValue == "" {
			continue
		}

		from, err := pomeriumAddress(fromValue)
		if err != nil {
			return nil, fmt.Errorf("parse pomerium route from %q in %q: %w", fromValue, config.Path(), err)
		}

		toValues := pomeriumToValues(configRoute.To)
		if len(toValues) == 0 {
			routes = append(routes, Route{
				Provider: ProviderNamePomerium,
				From:     from,
			})
			continue
		}

		for _, toValue := range toValues {
			to, perr := pomeriumAddress(stripPomeriumToWeight(toValue))
			if perr != nil {
				return nil, fmt.Errorf("parse pomerium route to %q in %q: %w", toValue, config.Path(), err)
			}

			routes = append(routes, Route{
				Provider: ProviderNamePomerium,
				From:     from,
				To:       &to,
			})
		}
	}

	return routes, nil
}

func isPomeriumYAMLConfig(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func pomeriumToValues(node yaml.Node) []string {
	switch node.Kind { //nolint:exhaustive // other kins not supported
	case yaml.ScalarNode:
		value := strings.TrimSpace(node.Value)
		if value == "" {
			return nil
		}

		return []string{value}
	case yaml.SequenceNode:
		values := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				continue
			}

			values = append(values, item.Value)
		}

		return values
	default:
		return nil
	}
}

func pomeriumAddress(value string) (Address, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return Address{}, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return Address{}, fmt.Errorf("expected absolute URL with scheme and host")
	}

	return Address{
		Address: pomeriumURLAddress(parsed),
		Domain:  parsed.Hostname(),
		Port:    parsed.Port(),
	}, nil
}

func pomeriumURLAddress(parsed *url.URL) string {
	address := parsed.Host
	if parsed.EscapedPath() != "" {
		address += parsed.EscapedPath()
	}
	if parsed.RawQuery != "" {
		address += "?" + parsed.RawQuery
	}

	return address
}

func stripPomeriumToWeight(value string) string {
	value = strings.TrimSpace(value)
	separator := strings.LastIndex(value, ",")
	if separator == -1 {
		return value
	}

	weight := strings.TrimSpace(value[separator+1:])
	if weight == "" {
		return value
	}
	if _, err := strconv.Atoi(weight); err != nil {
		return value
	}

	return strings.TrimSpace(value[:separator])
}
