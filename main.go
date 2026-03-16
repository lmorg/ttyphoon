package main

import (
	"fmt"
	"log"
	"os"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/utils/cache"
)

func main() {
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
