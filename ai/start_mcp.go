package ai

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/mxtty/ai/mcp_config"
	_ "github.com/lmorg/mxtty/ai/tools"
	"github.com/lmorg/mxtty/app"
)

func init() {
	/*err := mcp_config.StartServersFromJson("./mcp.json")
	if err != nil {
		log.Println(err)
	}*/

	for _, dir := range xdg.ConfigDirs {

		configPath := fmt.Sprintf("%s/%s/*.json", dir, strings.ToLower(app.Name))
		log.Println(configPath)

		files, err := filepath.Glob(configPath)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(files)

		for i := range files {
			log.Println(files[i])
			err = mcp_config.StartServersFromJson(files[i])
			if err != nil {
				log.Println(err)
			}
		}
	}
}
