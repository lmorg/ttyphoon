package ai

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/mcp"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

func StartMcp(renderer types.Renderer, meta *agent.Meta, cancel types.MenuCallbackT) {
	files := config.GetFiles("mcp", ".json")
	load := func(i int) {
		go func() {
			err := StartServersFromJson(renderer, meta, files[i])
			if err != nil {
				renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Cannot start MCP server from %s: %v", files[i], err))
			}
		}()
		StartMcp(renderer, meta, cancel)
	}

	renderer.DisplayMenu("Select a config file to load", files, nil, load, cancel)
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
		sticky := renderer.DisplaySticky(types.NOTIFY_INFO, fmt.Sprintf("Starting MCP server: %s", name), func() {})
		envs := svr.Env.Slice()

		if err = updateVars(meta, envs, cache); err != nil {
			sticky.Close()
			return err
		}
		if err = updateVars(meta, svr.Args, cache); err != nil {
			sticky.Close()
			return err
		}

		err = mcp.StartServerCmdLine(config.Source, meta, envs, name, svr.Command, svr.Args...)
		sticky.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

var (
	rxInput = regexp.MustCompile(`\$\{input:([-_a-zA-Z0-9]+)\}`)
	rxVars  = regexp.MustCompile(`\$\{([-_a-zA-Z0-9]+)\}`)
)

func updateVars(meta *agent.Meta, s []string, cache *map[string]string) error {
	var err error
	for i := range s {
		s[i], err = _updateVarsRxReplace(meta, s[i], cache)
		if err != nil {
			return err
		}
	}

	return nil
}

const _VAR_WORKSPACE_FOLDER = "workspaceFolder"

func _updateVarsRxReplace(meta *agent.Meta, s string, cache *map[string]string) (string, error) {
	var (
		val string
		ok  bool
	)

	match := rxInput.FindAllStringSubmatch(s, -1)
	for i := range match {
		val, ok = (*cache)[match[i][1]]
		if !ok {
			return "", fmt.Errorf("input missing: '%s'", match[i][1])
		}
		s = strings.ReplaceAll(s, match[i][0], val)
	}

	match = rxVars.FindAllStringSubmatch(s, -1)
	for i := range match {
		switch match[i][1] {
		case _VAR_WORKSPACE_FOLDER:
			if meta.Pwd == "" {
				return "", fmt.Errorf("unable to set ${%s} because pwd is unknown", _VAR_WORKSPACE_FOLDER)
			}
			val = meta.Pwd
		default:
			return "", fmt.Errorf("variable does not exist: '%s'", match[i][1])
		}
		s = strings.ReplaceAll(s, match[i][0], val)
	}

	return s, nil
}
