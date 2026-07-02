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

	validate(Languages)
}

func Append(conf []*LanguageBindingT) {
	validate(conf)
	Languages = append(Languages, conf...)
}

func validate(conf []*LanguageBindingT) {
	for _, binding := range conf {
		if binding.PathRegexp != "" {
			binding.pathRegexp = regexp.MustCompile(binding.PathRegexp)
		}
	}
}
