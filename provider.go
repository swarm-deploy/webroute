package webroute

import (
	"github.com/swarm-deploy/webroute/api"
	"github.com/swarm-deploy/webroute/providers"
)

func Providers() []api.Provider {
	return []api.Provider{
		providers.NewNginxProxyProvider(),
		providers.NewPomeriumProvider(),
		providers.NewAgentgatewayProvider(),
	}
}
