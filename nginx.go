package webroute

import (
	"fmt"
	"strings"
)

const (
	nginxVirtualHostKey = "VIRTUAL_HOST"
	nginxVirtualPathKey = "VIRTUAL_PATH"
	nginxVirtualPortKey = "VIRTUAL_PORT"
)

// NginxProxyProvider resolves routes configured for nginx-proxy.
type NginxProxyProvider struct{}

func NewNginxProxyProvider() *NginxProxyProvider {
	return &NginxProxyProvider{}
}

// Resolve resolves nginx-proxy routes from env values.
func (*NginxProxyProvider) Resolve(service Service) ([]Route, error) {
	env, err := service.Environment()
	if err != nil {
		return nil, fmt.Errorf("get environment variables: %w", err)
	}

	if len(env) == 0 {
		return nil, nil
	}

	virtualHosts := strings.TrimSpace(env[nginxVirtualHostKey])
	if virtualHosts == "" {
		return nil, nil
	}

	virtualPath := normalizeNginxPath(env[nginxVirtualPathKey])
	virtualPort := strings.TrimSpace(env[nginxVirtualPortKey])

	routes := make([]Route, 0)
	for _, host := range strings.Split(virtualHosts, ",") {
		domain := strings.TrimSpace(host)
		if domain == "" {
			continue
		}

		routes = append(routes, Route{
			Domain:  domain,
			Address: domain + "/" + virtualPath,
			Port:    virtualPort,
		})
	}

	return routes, nil
}

func normalizeNginxPath(path string) string {
	normalized := strings.TrimSpace(path)
	normalized = strings.TrimPrefix(normalized, "/")

	return normalized
}
