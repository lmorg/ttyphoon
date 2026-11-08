package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func syncPackageLicense(configPath, nfpmPath string) error {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var config struct {
		Info struct {
			License string `yaml:"license"`
		} `yaml:"info"`
	}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	license := config.Info.License
	if strings.TrimSpace(license) == "" || strings.ContainsAny(license, "\x00\r\n") {
		return fmt.Errorf("info.license must be a non-empty, single-line value")
	}
	data, err := os.ReadFile(nfpmPath)
	if err != nil {
		return fmt.Errorf("read nfpm config: %w", err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("parse nfpm config: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("nfpm config must contain a mapping")
	}
	mapping := document.Content[0]
	var value *yaml.Node
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == "license" {
			value = mapping.Content[i+1]
			break
		}
	}
	if value == nil || value.Kind != yaml.ScalarNode {
		return fmt.Errorf("nfpm config must contain a scalar license field")
	}
	value.Value = license
	value.Tag = "!!str"
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return fmt.Errorf("encode nfpm config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close nfpm encoder: %w", err)
	}
	if err := os.WriteFile(nfpmPath, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write nfpm config: %w", err)
	}
	return nil
}
