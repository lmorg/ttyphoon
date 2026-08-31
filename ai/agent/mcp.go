package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/mcp_config"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/file"
)

const mcpServerKeySeparator = "::"

func makeMcpServerKey(source, server string) string {
	return source + mcpServerKeySeparator + server
}

func parseMcpServerKey(key string) (source, server string, err error) {
	parts := strings.SplitN(strings.TrimSpace(key), mcpServerKeySeparator, 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid MCP server key")
	}

	return parts[0], parts[1], nil
}

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

func (agent *Agent) StartTools(tools []*skills.ToolsT) error {
	var err error
	for _, tool := range tools {
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
	cache := &map[string]string{}

	for i := range config.Mcp.Inputs {
		val, err := config.Mcp.Inputs[i].Get(agent.renderer)
		if err != nil {
			return err
		}
		(*cache)[config.Mcp.Inputs[i].Id] = val
	}

	for name, svr := range config.Mcp.Servers {
		if err := agent.startServerFromConfigEntry(config.Source, name, svr, cache); err != nil {
			return err
		}
	}

	return nil
}

func (agent *Agent) StartServerFromConfigServer(config *mcp_config.ConfigT, server string) error {
	if config == nil {
		return fmt.Errorf("MCP config is nil")
	}

	cache := &map[string]string{}
	for i := range config.Mcp.Inputs {
		val, err := config.Mcp.Inputs[i].Get(agent.renderer)
		if err != nil {
			return err
		}
		(*cache)[config.Mcp.Inputs[i].Id] = val
	}

	svr, ok := config.Mcp.Servers[server]
	if !ok {
		return fmt.Errorf("MCP server %q not found in %s", server, config.Source)
	}

	return agent.startServerFromConfigEntry(config.Source, server, svr, cache)
}

func (agent *Agent) startServerFromConfigEntry(source, name string, svr mcp_config.ServerT, cache *map[string]string) error {
	if agent.McpServerExists(name) {
		return nil
	}

	sticky := agent.renderer.DisplaySticky(types.NOTIFY_INFO, fmt.Sprintf("Starting MCP server: %s", name), func() {})
	defer sticky.Close()

	envs := svr.Env.Slice()

	var err error
	if err = updateVars(agent, envs, cache); err != nil {
		return err
	}
	if err = updateVars(agent, svr.Args, cache); err != nil {
		return err
	}
	if svr.Url != "" {
		svr.Url, err = _updateVarsRxReplace(agent, svr.Url, cache)
		if err != nil {
			return err
		}
	}
	if svr.OAuth != nil {
		svr.OAuth.ClientID, err = _updateVarsRxReplace(agent, svr.OAuth.ClientID, cache)
		if err != nil {
			return err
		}
		svr.OAuth.ClientURI, err = _updateVarsRxReplace(agent, svr.OAuth.ClientURI, cache)
		if err != nil {
			return err
		}
		svr.OAuth.ClientSecret, err = _updateVarsRxReplace(agent, svr.OAuth.ClientSecret, cache)
		if err != nil {
			return err
		}
		svr.OAuth.RedirectURI, err = _updateVarsRxReplace(agent, svr.OAuth.RedirectURI, cache)
		if err != nil {
			return err
		}
		svr.OAuth.AuthServerMetadataURL, err = _updateVarsRxReplace(agent, svr.OAuth.AuthServerMetadataURL, cache)
		if err != nil {
			return err
		}
		svr.OAuth.TokenFile, err = _updateVarsRxReplace(agent, svr.OAuth.TokenFile, cache)
		if err != nil {
			return err
		}
		if err = updateVars(agent, svr.OAuth.Scopes, cache); err != nil {
			return err
		}
	}

	switch svr.Type {
	case "http", "https":
		return startServerHttp(source, agent, name, svr)
	default:
		return startServerCmdLine(source, agent, envs, name, svr)
	}
}

func (agent *Agent) ListMcpServers() []map[string]interface{} {
	files := file.GetConfigFiles("mcp", ".json")
	sort.Strings(files)

	servers := make([]map[string]interface{}, 0)
	for _, cfgPath := range files {
		config, err := mcp_config.ReadJson(cfgPath)
		if err != nil {
			servers = append(servers, map[string]interface{}{
				"key":         makeMcpServerKey(cfgPath, ""),
				"name":        filepath.Base(cfgPath),
				"source":      cfgPath,
				"loaded":      false,
				"loadable":    false,
				"config":      "{}",
				"configError": err.Error(),
			})
			continue
		}

		for name, svr := range config.Mcp.Servers {
			cfgJSON, marshalErr := json.MarshalIndent(svr, "", "  ")
			if marshalErr != nil {
				cfgJSON = []byte("{}")
			}

			loadedSource := agent.McpServerSource(name)
			loaded := loadedSource != "" && loadedSource == cfgPath
			loadedElsewhere := loadedSource != "" && loadedSource != cfgPath

			servers = append(servers, map[string]interface{}{
				"key":               makeMcpServerKey(cfgPath, name),
				"name":              name,
				"source":            cfgPath,
				"loaded":            loaded,
				"loadable":          !loadedElsewhere,
				"loadedFrom":        loadedSource,
				"loadedElsewhere":   loadedElsewhere,
				"serverType":        svr.Type,
				"url":               svr.Url,
				"command":           svr.Command,
				"args":              append([]string{}, svr.Args...),
				"config":            string(cfgJSON),
				"configDisplayName": fmt.Sprintf("%s (%s)", name, filepath.Base(cfgPath)),
			})
		}
	}

	sort.Slice(servers, func(i, j int) bool {
		left := fmt.Sprintf("%s::%s", servers[i]["name"], servers[i]["source"])
		right := fmt.Sprintf("%s::%s", servers[j]["name"], servers[j]["source"])
		return left < right
	})

	return servers
}

func (agent *Agent) SetMcpServerEnabled(serverKey string, enabled bool) error {
	source, server, err := parseMcpServerKey(serverKey)
	if err != nil {
		return err
	}

	loadedSource := agent.McpServerSource(server)

	if enabled {
		if loadedSource == source {
			return nil
		}
		if loadedSource != "" && loadedSource != source {
			return fmt.Errorf("MCP server %q is already loaded from %s", server, loadedSource)
		}

		config, err := mcp_config.ReadJson(source)
		if err != nil {
			return err
		}
		config.Source = source

		if err := agent.StartServerFromConfigServer(config, server); err != nil {
			return err
		}

		agent.Reload()
		return nil
	}

	if loadedSource == "" {
		return nil
	}
	if loadedSource != source {
		return fmt.Errorf("MCP server %q is loaded from %s", server, loadedSource)
	}

	client, ok := agent._mcpServers[server]
	if !ok || client == nil {
		agent.McpServerRemove(server)
		return nil
	}

	if err := client.Close(); err != nil {
		return err
	}

	agent.McpServerRemove(server)

	agent.toolMu.Lock()
	nextTools := make([]aitypes.Tool, 0, len(agent._tools))
	for _, tool := range agent._tools {
		mcp, ok := tool.(*mcpTool)
		if ok && mcp.server == server {
			continue
		}
		nextTools = append(nextTools, tool)
	}
	agent._tools = nextTools
	agent.toolMu.Unlock()
	agent.Reload()

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
