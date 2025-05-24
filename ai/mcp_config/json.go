package mcp_config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type config struct {
	Mcp struct {
		Servers ServersT `json:"servers"`
		Inputs  InputsT  `json:"inputs"`
	} `json:"mcp"`
}

type ServersT map[string]ServerT

type ServerT struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     EnvVarsT `json:"env"`
}

type EnvVarsT map[string]string

func (env EnvVarsT) Slice() []string {
	var envvars []string
	for k, v := range env {
		envvars = append(envvars, fmt.Sprintf("%s=%s", k, v))
	}
	return envvars
}

func ReadJson(filename string) (*config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	config := new(config)
	err = json.Unmarshal(b, config)
	return config, err
}
