package jupyter

import (
	"bytes"
	_ "embed"
	"regexp"

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

	for _, binding := range Languages {
		if binding.PathRegexp != "" {
			binding.pathRegexp = regexp.MustCompile(binding.PathRegexp)
		}
	}
}
