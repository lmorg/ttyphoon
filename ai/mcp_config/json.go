package mcp_config

import (
	"encoding/json"
	"io"
	"os"

	mcp "github.com/lmorg/mxtty/ai/mcp2"
)

type ServerT struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type Servers struct {
	Servers map[string]ServerT `json:"servers"`
}

func readJson(filename string) (*Servers, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	servers := new(Servers)

	err = json.Unmarshal(b, servers)
	return servers, err
}

func StartServersFromJson(filename string) error {
	servers, err := readJson(filename)
	if err != nil {
		return err
	}

	for name, svr := range servers.Servers {
		err = mcp.StartServerCmdLine(name, svr.Command, svr.Args...)
		if err != nil {
			return err
		}
	}

	return nil
}
