package webroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-deploy/webroute/providers"
)

func TestProviders(t *testing.T) {
	providersList := Providers()
	assert.Len(t, providersList, 3)

	assert.Equal(t, []any{
		&providers.NginxProxyProvider{},
		&providers.PomeriumProvider{},
		&providers.AgentgatewayProvider{},
	}, []any{
		providersList[0],
		providersList[1],
		providersList[2],
	})
}
