package webroute

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-deploy/webroute/api"
	"github.com/swarm-deploy/webroute/providers"
)

func TestPomeriumProvider_Resolve(t *testing.T) {
	tests := []struct {
		Title    string
		Service  api.Service
		Expected []api.WebRoute
	}{
		{
			Title: "routes from yaml config",
			Service: &testService{configs: []api.ServiceConfig{
				testConfig{
					path: "/etc/pomerium/config.yaml",
					body: `
authenticate_service_url: https://authenticate.example.com
routes:
  - from: https://app.example.com
    to: http://app:8080
    policy:
      - allow:
          or:
            - email:
                is: user@example.com
  - from: https://admin.example.com:8443
    to: https://admin:9443/internal
`,
				},
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNamePomerium,
					From: api.Address{
						Domain:  "app.example.com",
						Address: "app.example.com",
					},
					To: &api.Address{
						Domain:  "app",
						Address: "app:8080",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNamePomerium,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com:8443",
						Port:    "8443",
					},
					To: &api.Address{
						Domain:  "admin",
						Address: "admin:9443/internal",
						Port:    "9443",
					},
				},
			},
		},
		{
			Title: "ignores non yaml configs and supports upstream list",
			Service: &testService{configs: []api.ServiceConfig{
				testConfig{
					path: "/etc/pomerium/plain.conf",
					body: "this is not yaml: :",
				},
				testConfig{
					path: "/etc/pomerium/routes.yml",
					body: `
routes:
  - from: https://weighted.example.com
    to:
      - http://app-a:8080,10
      - http://app-b:8080
`,
				},
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNamePomerium,
					From: api.Address{
						Domain:  "weighted.example.com",
						Address: "weighted.example.com",
					},
					To: &api.Address{
						Domain:  "app-a",
						Address: "app-a:8080",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNamePomerium,
					From: api.Address{
						Domain:  "weighted.example.com",
						Address: "weighted.example.com",
					},
					To: &api.Address{
						Domain:  "app-b",
						Address: "app-b:8080",
						Port:    "8080",
					},
				},
			},
		},
	}

	provider := providers.NewPomeriumProvider()

	for _, test := range tests {
		t.Run(test.Title, func(t *testing.T) {
			got, err := provider.Resolve(context.Background(), test.Service)
			require.NoError(t, err)

			assert.Equal(t, test.Expected, got)
		})
	}
}

func TestPomeriumProvider_ResolvePassesContextToConfigRead(t *testing.T) {
	type contextKey struct{}

	provider := providers.NewPomeriumProvider()
	ctx := context.WithValue(context.Background(), contextKey{}, "caller context")
	service := &testService{configs: []api.ServiceConfig{
		testConfig{
			path: "/etc/pomerium/config.yaml",
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
