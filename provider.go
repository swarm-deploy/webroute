package webroute

// Provider resolves web routes for a specific reverse proxy from env values.
type Provider interface {
	// Resolve resolves routes from normalized environment map.
	Resolve(service Service) ([]Route, error)
}

type Service interface {
	// Environment get environment variables
	Environment() (map[string]string, error)
}

func Providers() []Provider {
	return []Provider{
		NewNginxProxyProvider(),
	}
}
