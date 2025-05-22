package ai

import (
	"os"

	"github.com/lmorg/mxtty/ai/mcp_config"
	_ "github.com/lmorg/mxtty/ai/tools"
	"github.com/lmorg/mxtty/debug"
)

func init() {
	err := mcp_config.StartServersFromJson("mcp.json")
	if err != nil {
		debug.Log(err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		debug.Log(err)
		return
	}

	err = mcp_config.StartServersFromJson(home + "/mcp.json")
	if err != nil {
		debug.Log(err)
	}
}
