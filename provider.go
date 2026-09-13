package webroute

import (
	"context"
	"io"
)

type ProviderName string

const (
	ProviderNameAgentgateway ProviderName = "agentgateway"
	ProviderNameNginxProxy   ProviderName = "nginx-proxy"
	ProviderNamePomerium     ProviderName = "pomerium"
)

// Provider resolves web routes for a specific reverse proxy from env values.
type Provider interface {
	// Resolve resolves routes from normalized environment map.
	Resolve(ctx context.Context, service Service) ([]Route, error)
}

type Service interface {
	// Environment get environment variables
	Environment() (map[string]string, error)
	Configs() []ServiceConfig
}

type ServiceConfig interface {
	Path() string
	Read(ctx context.Context, out io.Writer) error
}

func Providers() []Provider {
	return []Provider{
		NewNginxProxyProvider(),
		NewPomeriumProvider(),
		NewAgentgatewayProvider(),
	}
}
