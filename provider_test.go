package webroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-deploy/webroute/providers/agentgateway"
	"github.com/swarm-deploy/webroute/providers/nginxproxy"
	"github.com/swarm-deploy/webroute/providers/pomerium"
)

func TestProviders(t *testing.T) {
	providersList := Providers()
	assert.Len(t, providersList, 3)

	assert.Equal(t, []any{
		&nginxproxy.NginxProxyProvider{},
		&pomerium.PomeriumProvider{},
		&agentgateway.AgentgatewayProvider{},
	}, []any{
		providersList[0],
		providersList[1],
		providersList[2],
	})
}
