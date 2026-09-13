package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-deploy/webroute/api"
)

func TestNginxProxyProvider_Resolve(t *testing.T) {
	tests := []struct {
		Title    string
		Service  api.Service
		Expected []api.WebRoute
	}{
		{
			Title: "basic test",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "api.example.com, admin.example.com",
				"VIRTUAL_PATH": "/v1",
				"VIRTUAL_PORT": "8080",
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "api.example.com",
						Address: "api.example.com/v1",
						Port:    "8080",
					},
				},
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com/v1",
						Port:    "8080",
					},
				},
			},
		},
		{
			Title: "without virtual path",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "app.example.com",
				"VIRTUAL_PORT": "80",
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "app.example.com",
						Address: "app.example.com/",
						Port:    "80",
					},
				},
			},
		},
		{
			Title: "root virtual path",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "app.example.com",
				"VIRTUAL_PATH": "/",
				"VIRTUAL_PORT": "80",
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "app.example.com",
						Address: "app.example.com/",
						Port:    "80",
					},
				},
			},
		},
		{
			Title: "virtual host with port",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "admin.example.com:8443",
				"VIRTUAL_PATH": "/admin",
				"VIRTUAL_PORT": "8080",
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "admin.example.com",
						Address: "admin.example.com:8443/admin",
						Port:    "8080",
					},
				},
			},
		},
		{
			Title: "virtual host multiports",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "ignored.example.com",
				"VIRTUAL_HOST_MULTIPORTS": `
www.example.org:
service1.example.org:
  "/":
    port: 8000
service2.example.org:
  "/api":
    port: "9000"
  "/healthz":
`,
			}},
			Expected: []api.WebRoute{
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "service1.example.org",
						Address: "service1.example.org/",
						Port:    "8000",
					},
				},
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "service2.example.org",
						Address: "service2.example.org/api",
						Port:    "9000",
					},
				},
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "service2.example.org",
						Address: "service2.example.org/healthz",
					},
				},
				{
					Provider: api.ProviderNameNginxProxy,
					From: api.Address{
						Domain:  "www.example.org",
						Address: "www.example.org/",
					},
				},
			},
		},
	}

	provider := NewNginxProxyProvider()

	for _, test := range tests {
		t.Run(test.Title, func(t *testing.T) {
			got, err := provider.Resolve(context.Background(), test.Service)
			require.NoError(t, err)

			assert.Equal(t, test.Expected, got)
		})
	}
}
