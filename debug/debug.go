package debug

import (
	"encoding/json"
	"log"
	"runtime"
	"strings"

	"github.com/lmorg/ttyphoon/app"
	_ "github.com/lmorg/ttyphoon/debug/pprof"
)

func Log(v any) {
	if !Enabled {
		return
	}

	var (
		b   []byte
		err error
	)

	switch t := v.(type) {
	case byte:
		v = string(t)

	case []byte:
		v = string(t)

	case []rune:
		if len(t) > 0 && t[0] < 32 {
			break
		}
		v = string(t)

	case string:
		b = []byte(t)
		goto skipJson

	}

	b, err = json.MarshalIndent(v, "", "    ")
	if err != nil {
		panic(err)
	}

skipJson:

	pc, file, line, ok := runtime.Caller(1)

	if !ok {
		log.Printf("DEBUG: %s:%d: %s", file, line, string(b))
		return
	}

	fn := runtime.FuncForPC(pc)
	fnName := strings.Replace(fn.Name(), app.ProjectSourcePath, "", 1)

	pc, _, _, ok = runtime.Caller(2)
	if !ok {
		log.Printf("DEBUG: %s(): %s", fnName, string(b))
		return
	}

	fn = runtime.FuncForPC(pc)
	prevName := strings.Replace(fn.Name(), app.ProjectSourcePath, "", 1)

	s := strings.ReplaceAll(string(b), `\n`, "\n")
	lines := strings.Split(s, "\n")
	for i := range lines {
		log.Printf("DEBUG: %s() -> %s(): %s", prevName, fnName, lines[i])
	}
}
