package ai

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/ai/mcp"
	"github.com/lmorg/mxtty/ai/mcp_config"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
)

func StartMcp(renderer types.Renderer, meta *agent.Meta) {
	files := config.GetFiles("mcp", ".json")
	load := func(i int) {
		go func() {
			err := StartServersFromJson(renderer, meta, files[i])
			if err != nil {
				renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Cannot start MCP server from %s: %v", files[i], err))
			}
		}()
	}

	renderer.DisplayMenu("Select a config file to load", files, nil, load, nil)
}

func StartServersFromJson(renderer types.Renderer, meta *agent.Meta, filename string) error {
	config, err := mcp_config.ReadJson(filename)
	if err != nil {
		return err
	}
	config.Source = filename
	return StartServersFromConfig(renderer, meta, config)
}

func StartServersFromConfig(renderer types.Renderer, meta *agent.Meta, config *mcp_config.ConfigT) error {
	var err error
	cache := &map[string]string{}

	for i := range config.Mcp.Inputs {
		val, err := config.Mcp.Inputs[i].Get(renderer)
		if err != nil {
			return err
		}
		(*cache)[config.Mcp.Inputs[i].Id] = val
	}

	for name, svr := range config.Mcp.Servers {
		if meta.McpServerExists(name) {
			renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Skipping MCP server '%s': a server with the same name is already running", name))
			continue
		}
		renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Starting MCP server: %s", name))

		envs := svr.Env.Slice()

		updateVars(envs, cache)
		updateVars(svr.Args, cache)

		err = mcp.StartServerCmdLine(config.Source, meta, envs, name, svr.Command, svr.Args...)
		if err != nil {
			return err
		}
	}

	return nil
}

var rxInput = regexp.MustCompile(`\$\{input:([-_a-zA-Z0-9]+)\}`)

func updateVars(s []string, cache *map[string]string) {
	for i := range s {
		s[i] = _updateVarsRxReplace(s[i], cache)
	}
}

func _updateVarsRxReplace(s string, cache *map[string]string) string {
	match := rxInput.FindAllStringSubmatch(s, -1)
	for i := range match {
		s = strings.ReplaceAll(s, match[i][0], (*cache)[match[i][1]])
	}
	return s
}
