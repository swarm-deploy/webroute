package agentgateway

import (
	"context"
	"io"
	"strings"

	"github.com/swarm-deploy/webroute/api"
)

type testService struct {
	environment map[string]string
	configs     []api.ServiceConfig
}

func (s *testService) Environment() (map[string]string, error) {
	return s.environment, nil
}

func (s *testService) Configs() []api.ServiceConfig {
	return s.configs
}

type testConfig struct {
	path string
	body string
	read func(context.Context, io.Writer) error
}

func (c testConfig) Path() string {
	return c.path
}

func (c testConfig) Read(ctx context.Context, out io.Writer) error {
	if c.read != nil {
		return c.read(ctx, out)
	}

	_, err := io.Copy(out, strings.NewReader(c.body))
	return err
}
