package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var versionLinePattern = regexp.MustCompile(`^([ \t]*version:[ \t]*)(['"]?)([^'"#\s]+)(['"]?)([ \t]*(?:#.*)?)(\r?)$`)

func incrementConfigVersion(configPath string) (string, string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", fmt.Errorf("read config %s: %w", configPath, err)
	}
	updated, previous, next, err := incrementVersion(data)
	if err != nil {
		return "", "", err
	}
	info, err := os.Stat(configPath)
	if err != nil {
		return "", "", fmt.Errorf("stat config %s: %w", configPath, err)
	}
	if err := os.WriteFile(configPath, updated, info.Mode().Perm()); err != nil {
		return "", "", fmt.Errorf("write config %s: %w", configPath, err)
	}
	return previous, next, nil
}

func incrementVersion(data []byte) ([]byte, string, string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, "", "", fmt.Errorf("parse config: %w", err)
	}
	root := mappingValue(&document, "info")
	versionNode := mappingValue(root, "version")
	if versionNode == nil || versionNode.Kind != yaml.ScalarNode || strings.TrimSpace(versionNode.Value) == "" {
		return nil, "", "", fmt.Errorf("info.version must be a three-part numeric version")
	}

	parts := strings.Split(versionNode.Value, ".")
	if len(parts) != 3 {
		return nil, "", "", fmt.Errorf("info.version %q must be a three-part numeric version", versionNode.Value)
	}
	for _, part := range parts {
		if part == "" {
			return nil, "", "", fmt.Errorf("info.version %q must be a three-part numeric version", versionNode.Value)
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return nil, "", "", fmt.Errorf("info.version %q must be a three-part numeric version: %w", versionNode.Value, err)
		}
	}
	patch, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return nil, "", "", fmt.Errorf("parse info.version patch component: %w", err)
	}
	if patch == ^uint64(0) {
		return nil, "", "", fmt.Errorf("info.version patch component is too large to increment")
	}
	nextPatch := strconv.FormatUint(patch+1, 10)
	if len(nextPatch) < len(parts[2]) {
		nextPatch = strings.Repeat("0", len(parts[2])-len(nextPatch)) + nextPatch
	}
	next := parts[0] + "." + parts[1] + "." + nextPatch

	lines := strings.Split(string(data), "\n")
	lineIndex := versionNode.Line - 1
	if lineIndex < 0 || lineIndex >= len(lines) {
		return nil, "", "", fmt.Errorf("info.version points to an invalid config line")
	}
	matches := versionLinePattern.FindStringSubmatch(lines[lineIndex])
	if matches == nil || matches[2] != matches[4] || matches[3] != versionNode.Value {
		return nil, "", "", fmt.Errorf("cannot safely update info.version in config")
	}
	lines[lineIndex] = matches[1] + matches[2] + next + matches[4] + matches[5] + matches[6]
	return []byte(strings.Join(lines, "\n")), versionNode.Value, next, nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil
		}
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}
