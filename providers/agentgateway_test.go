package providers

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-deploy/webroute/api"
)

func TestAgentgatewayProvider_Resolve(t *testing.T) {
	tests := []struct {
		Title    string
		Service  api.Service
		Expected []api.WebRoute
	}{
		{
			Title: "routes from yaml config",
			Service: &testService{configs: []api.ServiceConfig{
				testConfig{
					path: "/etc/agentgateway/config.yaml",
					body: `
gateways:
  default:
    port: 3000
routes:
  - name: api
    hostnames:
      - api.example.com
      - admin.example.com:8443
    matches:
      - path:
          pathPrefix: /api
      - path:
          exact: /healthz
    backends:
      - host: api:8080
      - host: http://api-canary:8081/internal
`,
				},
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "api.example.com",
						Address: "api.example.com:3000/api",
						Port:    "3000",
					},
					To: &api.Address{
						Domain:  "api",
						Address: "api:8080",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "api.example.com",
						Address: "api.example.com:3000/api",
						Port:    "3000",
					},
					To: &api.Address{
						Domain:  "api-canary",
						Address: "api-canary:8081/internal",
						Port:    "8081",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "api.example.com",
						Address: "api.example.com:3000/healthz",
						Port:    "3000",
					},
					To: &api.Address{
						Domain:  "api",
						Address: "api:8080",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "api.example.com",
						Address: "api.example.com:3000/healthz",
						Port:    "3000",
					},
					To: &api.Address{
						Domain:  "api-canary",
						Address: "api-canary:8081/internal",
						Port:    "8081",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com:8443/api",
						Port:    "8443",
					},
					To: &api.Address{
						Domain:  "api",
						Address: "api:8080",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com:8443/api",
						Port:    "8443",
					},
					To: &api.Address{
						Domain:  "api-canary",
						Address: "api-canary:8081/internal",
						Port:    "8081",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com:8443/healthz",
						Port:    "8443",
					},
					To: &api.Address{
						Domain:  "api",
						Address: "api:8080",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com:8443/healthz",
						Port:    "8443",
					},
					To: &api.Address{
						Domain:  "api-canary",
						Address: "api-canary:8081/internal",
						Port:    "8081",
					},
				},
			},
		},
		{
			Title: "ignores non yaml configs and supports route defaults",
			Service: &testService{configs: []api.ServiceConfig{
				testConfig{
					path: "/etc/agentgateway/plain.conf",
					body: "this is not yaml: :",
				},
				testConfig{
					path: "/etc/pomerium/config.yaml",
					body: `
routes:
  - from: https://pomerium.example.com
    to: http://app:8080
`,
				},
				testConfig{
					path: "/etc/agentgateway/routes.yml",
					body: `
routes:
  - name: catch-all
    backends:
      - host: localhost:8000
  - name: frontend
    hostnames: [www.example.com]
    backends:
      - host: web:3000
`,
				},
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Address: "/",
					},
					To: &api.Address{
						Domain:  "localhost",
						Address: "localhost:8000",
						Port:    "8000",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "www.example.com",
						Address: "www.example.com/",
					},
					To: &api.Address{
						Domain:  "web",
						Address: "web:3000",
						Port:    "3000",
					},
				},
			},
		},
		{
			Title: "route without backend",
			Service: &testService{configs: []api.ServiceConfig{
				testConfig{
					path: "/etc/agentgateway/config.yaml",
					body: `
routes:
  - name: direct-response
    hostnames: [status.example.com]
    matches:
      - path:
          regex: ^/status/[0-9]+$
`,
				},
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Domain:  "status.example.com",
						Address: "status.example.com/^/status/[0-9]+$",
					},
				},
			},
		},
		{
			Title: "mcp backend target host",
			Service: &testService{
				environment: map[string]string{
					"MCP_PUBLIC_PORT":   "443",
					"POSTGRES_MCP_HOST": "postgres-mcp-core",
				},
				configs: []api.ServiceConfig{
					testConfig{
						path: "/etc/agentgateway/config.yaml",
						body: `
frontendPolicies:
  tracing:
    host: infra-otel-collector:4317
    protocol: grpc
    randomSampling: true
gateways:
  default:
    port: ${MCP_PUBLIC_PORT}
routes:
  - name: postgres-mcp-db-core
    matches:
      - path:
          exact: /mcp/postgres/db/core
      - path:
          exact: /.well-known/oauth-protected-resource/mcp/postgres/db/core
    policies:
      authorization:
        rules:
          - require: 'has(jwt.scope) && "${MCP_REQUIRED_SCOPE}" in jwt.scope.split(" ")'
      mcpAuthentication:
        mode: strict
        resourceMetadata:
          resource: https://${MCP_PUBLIC_HOST}/mcp/postgres/db/core
    backends:
      - mcp:
          targets:
            - name: postgres
              mcp:
                host: http://${POSTGRES_MCP_HOST}:8000/mcp
`,
					},
				},
			},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Address: "/mcp/postgres/db/core",
						Port:    "443",
					},
					To: &api.Address{
						Domain:  "postgres-mcp-core",
						Address: "postgres-mcp-core:8000/mcp",
						Port:    "8000",
					},
				},
				{
					Provider: api.ProviderNameAgentgateway,
					From: api.Address{
						Address: "/.well-known/oauth-protected-resource/mcp/postgres/db/core",
						Port:    "443",
					},
					To: &api.Address{
						Domain:  "postgres-mcp-core",
						Address: "postgres-mcp-core:8000/mcp",
						Port:    "8000",
					},
				},
			},
		},
	}

	provider := NewAgentgatewayProvider()

	for _, test := range tests {
		t.Run(test.Title, func(t *testing.T) {
			got, err := provider.Resolve(context.Background(), test.Service)
			require.NoError(t, err)

			assert.Equal(t, test.Expected, got)
		})
	}
}

func TestAgentgatewayProvider_ResolvePassesContextToConfigRead(t *testing.T) {
	type contextKey struct{}

	provider := NewAgentgatewayProvider()
	ctx := context.WithValue(context.Background(), contextKey{}, "caller context")
	service := &testService{configs: []api.ServiceConfig{
		testConfig{
			path: "/etc/agentgateway/config.yaml",
			read: func(ctx context.Context, out io.Writer) error {
				assert.Equal(t, "caller context", ctx.Value(contextKey{}))

				_, err := io.WriteString(out, "routes: []")
				return err
			},
		},
	}}

	_, err := provider.Resolve(ctx, service)
	require.NoError(t, err)
}
