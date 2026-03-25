package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/lmorg/ttyphoon/utils/file"
)

func main() {
	if runtime.GOOS == "darwin" {
		err := os.Setenv("PATH", "PATH="+os.Getenv("PATH")+":/usr/bin:/opt/homebrew/bin:/opt/homebrew/sbin")
		if err != nil {
			panic(err)
		}
	}

	loadEnvs()

	cacheDbFile := "cache.db"
	cacheDbPath, err := xdg.CacheFile(cacheDbFile)
	if err != nil {
		log.Println(err)
		cacheDbPath = fmt.Sprintf("%s/%s-%s", os.TempDir(), app.DirName, cacheDbFile)
	}
	cache.SetPath(cacheDbPath)
	cache.InitCache()

	startWails()
}

func loadEnvs() {
	files := file.GetConfigFiles("/", ".env")
	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Print(err)
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			split := strings.SplitN(scanner.Text(), "=", 2)
			if len(split) != 2 {
				split = []string{files[i], ""}
			}
			debug.Log(fmt.Sprintf(`%s: "%s" = "%s"`, files[i], split[0], split[1]))
			os.Setenv(split[0], split[1])
		}
	}
}

func cdHome() {
	// default to $HOME
	home, err := os.UserHomeDir()
	if err != nil {
		os.Stderr.WriteString(err.Error())

	} else {
		if err = os.Chdir(home); err != nil {
			os.Stderr.WriteString(err.Error())
		}
	}
}
