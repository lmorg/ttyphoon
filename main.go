package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/cache"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(app.Version())
		return
	}

	if runtime.GOOS == "darwin" {
		err := os.Setenv("PATH", "PATH="+os.Getenv("PATH")+":/usr/bin:/opt/homebrew/bin:/opt/homebrew/sbin")
		if err != nil {
			panic(err)
		}
	}

	config.ReadEnvConfig()

	cacheDbFile := "cache.db"
	cacheDbPath, err := xdg.CacheFile(cacheDbFile)
	if err != nil {
		log.Printf("[error] %v", err)
		cacheDbPath = fmt.Sprintf("%s/%s-%s", os.TempDir(), app.DirName, cacheDbFile)
	}
	cache.SetPath(cacheDbPath)
	cache.InitCache()

	startWails()
}
