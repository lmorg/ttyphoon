package jupyter

import (
	"bytes"
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed languages.yaml
var defaults []byte

func init() {
	buf := bytes.NewBuffer(defaults)
	yml := yaml.NewDecoder(buf)
	yml.KnownFields(true)

	err := yml.Decode(&Languages)
	if err != nil {
		panic(err)
	}
}
