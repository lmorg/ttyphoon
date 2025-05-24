package ai

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/mxtty/ai/mcp"
	"github.com/lmorg/mxtty/ai/mcp_config"
	_ "github.com/lmorg/mxtty/ai/tools"
	"github.com/lmorg/mxtty/app"
	"github.com/lmorg/mxtty/types"
)

func StartMcp(renderer types.Renderer) {
	for _, dir := range xdg.ConfigDirs {

		configPath := fmt.Sprintf("%s/%s/startup/*.json", dir, strings.ToLower(app.Name))
		log.Println(configPath)

		files, err := filepath.Glob(configPath)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(files)

		for i := range files {
			log.Println(files[i])
			err = startServersFromJson(files[i])
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func startServersFromJson(filename string) error {
	config, err := mcp_config.ReadJson(filename)
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
