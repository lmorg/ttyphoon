package main

import (
	"encoding/json"
	"os"

	"github.com/lmorg/ttyphoon/app"
)

func main() {
	if err := json.NewEncoder(os.Stdout).Encode([]string{
		app.Name(), app.TagLine(), app.Version(), app.Copyright(),
		app.Branch(), app.BuildDate(), app.License(),
	}); err != nil {
		panic(err)
	}
}
