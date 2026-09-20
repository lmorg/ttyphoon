package integrations

import (
	"embed"
	"fmt"
)

//go:embed shell.*
var embedFs embed.FS

var integrations map[string][]byte

func init() {
	integrations = make(map[string][]byte)

	dir, err := embedFs.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for i := range dir {
		name := dir[i].Name()

		b, err := embedFs.ReadFile(name)
		if err != nil {
			panic(err)
		}

		integrations[name] = b
	}
}

func Get(name string) []byte {
	b, ok := integrations[name]
	if !ok {
		panic(fmt.Sprintf("no asset found named '%s'", name))
	}
	return b
}
