package notes

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func listFiles(path string, varName string) (files []string) {
	return listFilesWithGlob(path, varName, "*")
}

func listFilesWithGlob(path string, varName string, pattern string) (files []string) {
	if path == "" {
		return []string{}
	}

	glob, err := filepath.Glob(fmt.Sprintf("%s/%s", path, pattern))
	if err != nil {
		log.Println(err)
		return
	}
	replace := fmt.Sprintf("$%s/", varName)
	for i := range glob {
		info, err := os.Stat(glob[i])
		if err != nil {
			continue
		}

		if info.IsDir() || isSystemFileName(info.Name()) {
			continue
		}

		files = append(files, strings.Replace(glob[i], path, replace, 1))
	}
	return
}

func isSystemFileName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "thumbs.db", "ehthumbs.db", "desktop.ini", ".ds_store":
		return true
	}

	// macOS AppleDouble sidecar files
	return strings.HasPrefix(name, "._")
}
