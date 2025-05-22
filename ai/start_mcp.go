package ai

import (
	"github.com/lmorg/mxtty/ai/mcp_config"
	_ "github.com/lmorg/mxtty/ai/tools"
)

func init() {
	err := mcp_config.StartServersFromJson("mcp.json")
	if err != nil {
		panic(err)
	}
}
