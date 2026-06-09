package syntaxcompletion

import (
	"bytes"
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed default_rules.yaml
var defaultRulesYAML []byte

type Config struct {
	Languages map[string]LanguageConfig `yaml:"languages"`
}

type LanguageConfig struct {
	Pairs            []PairRule            `yaml:"pairs"`
	ListContinuation *ListContinuationRule `yaml:"list_continuation"`
	AutoCloseTags    *AutoCloseTagsRule    `yaml:"auto_close_tags"`
	ContextGuard     *ContextGuardRule     `yaml:"context_guard"`
}

type PairRule struct {
	Open          string `yaml:"open"`
	Close         string `yaml:"close"`
	SkipClose     bool   `yaml:"skip_close"`
	WrapSelection bool   `yaml:"wrap_selection"`
}

type ListContinuationRule struct {
	Enabled          bool     `yaml:"enabled"`
	UnorderedMarkers []string `yaml:"unordered_markers"`
	Ordered          bool     `yaml:"ordered"`
	ExitOnEmptyItem  bool     `yaml:"exit_on_empty_item"`
}

type AutoCloseTagsRule struct {
	Enabled bool     `yaml:"enabled"`
	Allowed []string `yaml:"allowed"`
}

type ContextGuardRule struct {
	Enabled    bool     `yaml:"enabled"`
	DisallowIn []string `yaml:"disallow_in"`
}

func LoadConfig(yml []byte) (*Config, error) {
	cfg := &Config{}
	dec := yaml.NewDecoder(bytes.NewBuffer(yml))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, err
	}
	if cfg.Languages == nil {
		cfg.Languages = map[string]LanguageConfig{}
	}
	return cfg, nil
}

func LoadDefaultConfig() (*Config, error) {
	return LoadConfig(defaultRulesYAML)
}
