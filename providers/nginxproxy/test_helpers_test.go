package nginxproxy

import (
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
