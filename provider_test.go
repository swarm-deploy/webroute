package webroute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNginxProxyProvider_Resolve(t *testing.T) {
	tests := []struct {
		Title    string
		Service  Service
		Expected []Route
	}{
		{
			Title: "basic test",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "api.example.com, admin.example.com",
				"VIRTUAL_PATH": "/v1",
				"VIRTUAL_PORT": "8080",
			}},
			Expected: []Route{
				{
					Domain:  "api.example.com",
					Address: "api.example.com/v1",
					Port:    "8080",
				},
				{
					Domain:  "admin.example.com",
					Address: "admin.example.com/v1",
					Port:    "8080",
				},
			},
		},
		{
			Title: "without virtual path",
			Service: &testService{environment: map[string]string{
				"VIRTUAL_HOST": "app.example.com",
				"VIRTUAL_PORT": "80",
			}},
			Expected: []Route{
				{
					Domain:  "app.example.com",
					Address: "app.example.com/",
					Port:    "80",
				},
			},
		},
	}

	provider := NewNginxProxyProvider()

	for _, test := range tests {
		t.Run(test.Title, func(t *testing.T) {
			got, err := provider.Resolve(test.Service)
			require.NoError(t, err)

			assert.Equal(t, test.Expected, got)
		})
	}
}

type testService struct {
	environment map[string]string
}

func (s *testService) Environment() (map[string]string, error) {
	return s.environment, nil
}
