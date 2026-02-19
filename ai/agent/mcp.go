package agent

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
)

func (agent *Agent) McpMenu(cancel types.MenuCallbackT) {
	files := file.GetConfigFiles("mcp", ".json")
	load := func(i int) {
		go func() {
			err := agent.StartServersFromJson(files[i])
			if err != nil {
				agent.renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Cannot start MCP server from %s: %v", files[i], err))
			}
		}()
		agent.McpMenu(cancel)
	}

	agent.renderer.DisplayMenu("Select a config file to load", files, nil, load, cancel)
}

func (agent *Agent) SkillStartTools(skill *skills.SkillT) error {
	var err error
	for _, tool := range skill.Tools {
		switch tool.Name {
		case "mcp":
			var filename string
			filename, err = file.GetConfigFile("mcp", tool.Parameters+".json")
			if err != nil {
				return err
			}
			err = agent.StartServersFromJson(filename)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (agent *Agent) StartServersFromJson(filename string) error {
	config, err := mcp_config.ReadJson(filename)
	if err != nil {
		return err
	}
	config.Source = filename
	return agent.StartServersFromConfig(config)
}

func (agent *Agent) StartServersFromConfig(config *mcp_config.ConfigT) error {
	var err error
	cache := &map[string]string{}

	for i := range config.Mcp.Inputs {
		val, err := config.Mcp.Inputs[i].Get(agent.renderer)
		if err != nil {
			return err
		}
		(*cache)[config.Mcp.Inputs[i].Id] = val
	}

	for name, svr := range config.Mcp.Servers {
		if agent.McpServerExists(name) {
			//renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("Skipping MCP server '%s': a server with the same name is already running", name))
			continue
		}
		sticky := agent.renderer.DisplaySticky(types.NOTIFY_INFO, fmt.Sprintf("Starting MCP server: %s", name), func() {})
		envs := svr.Env.Slice()

		if err = updateVars(agent, envs, cache); err != nil {
			sticky.Close()
			return err
		}
		if err = updateVars(agent, svr.Args, cache); err != nil {
			sticky.Close()
			return err
		}

		switch svr.Type {
		case "http", "https":
			err = startServerHttp(config.Source, agent, name, svr.Url)
		default:
			err = startServerCmdLine(config.Source, agent, envs, name, svr.Command, svr.Args...)
		}
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

func updateVars(agent *Agent, s []string, cache *map[string]string) error {
	var err error
	for i := range s {
		s[i], err = _updateVarsRxReplace(agent, s[i], cache)
		if err != nil {
			return err
		}
	}

	return nil
}

const _VAR_WORKSPACE_FOLDER = "workspaceFolder"

func _updateVarsRxReplace(agent *Agent, s string, cache *map[string]string) (string, error) {
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
			if agent.Meta.Pwd == "" {
				return "", fmt.Errorf("unable to set ${%s} because pwd is unknown", _VAR_WORKSPACE_FOLDER)
			}
			val = agent.Meta.Pwd
		default:
			return "", fmt.Errorf("variable does not exist: '%s'", match[i][1])
		}
		s = strings.ReplaceAll(s, match[i][0], val)
	}

	return s, nil
}
