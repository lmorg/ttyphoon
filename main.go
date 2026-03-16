package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/utils/cache"
)

func main() {
	if runtime.GOOS == "darwin" {
		err := os.Setenv("PATH", "PATH="+os.Getenv("PATH")+":/usr/bin:/opt/homebrew/bin:/opt/homebrew/sbin")
		if err != nil {
			panic(err)
		}
	}

	loadEnvs()

	/*switch dispatcher.AppTypeT(os.Getenv(dispatcher.ENV_APP)) {
	case dispatcher.AppGlobalHotkeys:
		globalhotkeys.Register()
		return
	}*/

	cacheDbFile := "cache.db"
	cacheDbPath, err := xdg.CacheFile(cacheDbFile)
	if err != nil {
		log.Println(err)
		cacheDbPath = fmt.Sprintf("%s/%s-%s", os.TempDir(), app.DirName, cacheDbFile)
	}
	cache.SetPath(cacheDbPath)
	cache.InitCache()

	//defer dispatcher.CleanUp()

	startWails()
}
