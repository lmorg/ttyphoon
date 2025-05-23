package mcp_config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/lmorg/mxtty/ai/mcp"
)

type config struct {
	Mcp struct {
		Servers Servers `json:"servers"`
	} `json:"mcp"`
}

type Servers map[string]Server

type Server struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     envT     `json:"env"`
}

type envT map[string]string

func (env envT) Slice() []string {
	var envvars []string
	for k, v := range env {
		envvars = append(envvars, fmt.Sprintf("%s=%s", k, v))
	}
	return envvars
}

func readJson(filename string) (*config, error) {
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

func StartServersFromJson(filename string) error {
	config, err := readJson(filename)
	if err != nil {
		return err
	}

	for name, svr := range config.Mcp.Servers {
		err = mcp.StartServerCmdLine(svr.Env.Slice(), name, svr.Command, svr.Args...)
		if err != nil {
			return err
		}
	}

	return nil
}
