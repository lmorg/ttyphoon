package config

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/mxtty/app"
)

func GetFiles(subDir string, fileExt string) []string {
	var files []string

	for _, dir := range xdg.ConfigDirs {
		configPath := fmt.Sprintf("%s/%s/%s/*%s", dir, strings.ToLower(app.Name), subDir, fileExt)
		log.Println(configPath)

		matched, _ := filepath.Glob(configPath)
		files = append(files, matched...)
	}

	return files
}
