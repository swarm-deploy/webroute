package webroute

import (
	"github.com/swarm-deploy/webroute/api"
	"github.com/swarm-deploy/webroute/providers/agentgateway"
	"github.com/swarm-deploy/webroute/providers/nginxproxy"
	"github.com/swarm-deploy/webroute/providers/pomerium"
)

func Providers() []api.Provider {
	return []api.Provider{
		nginxproxy.NewNginxProxyProvider(),
		pomerium.NewPomeriumProvider(),
		agentgateway.NewAgentgatewayProvider(),
	}
}
