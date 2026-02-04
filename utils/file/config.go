package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
)

func GetConfigGlob(glob string) []string {
	var files []string

	for _, dir := range xdg.ConfigDirs {
		configPath := fmt.Sprintf("%s/%s/%s", dir, strings.ToLower(app.Name), glob)
		debug.Log(configPath)

		matched, _ := filepath.Glob(configPath)
		files = append(files, matched...)
	}

	return files
}

func GetConfigFiles(subDir string, fileExt string) []string {
	var files []string

	for _, dir := range xdg.ConfigDirs {
		configPath := fmt.Sprintf("%s/%s/%s/*%s", dir, strings.ToLower(app.Name), subDir, fileExt)
		debug.Log(configPath)

		matched, _ := filepath.Glob(configPath)
		files = append(files, matched...)
	}

	return files
}

func GetConfigFile(subDir string, resourceName string) (string, error) {
	for _, dir := range xdg.ConfigDirs {
		filename := fmt.Sprintf("%s/%s/%s/%s", dir, strings.ToLower(app.Name), subDir, resourceName)
		debug.Log(filename)

		if Exists(filename) {
			return filename, nil
		}
	}

	return "", fmt.Errorf("cannot find file: %s", resourceName)
}

func OpenConfigFile(subDir string, resourceName string) (*os.File, error) {
	for _, dir := range xdg.ConfigDirs {
		filename := fmt.Sprintf("%s/%s/%s/%s", dir, strings.ToLower(app.Name), subDir, resourceName)
		debug.Log(filename)

		if Exists(filename) {
			return os.Open(filename)
		}
	}

	return nil, fmt.Errorf("cannot find file: %s", resourceName)
}
