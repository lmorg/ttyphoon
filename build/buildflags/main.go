package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", "build/config.yml", "Wails configuration file")
	nfpmPath := flag.String("nfpm", "", "update the license in this nfpm file instead of printing linker flags")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "build flags: unexpected positional arguments")
		os.Exit(1)
	}
	if *nfpmPath != "" {
		if err := syncPackageLicense(*configPath, *nfpmPath); err != nil {
			fmt.Fprintln(os.Stderr, "package license:", err)
			os.Exit(1)
		}
		return
	}
	flags, err := buildFlags(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "build flags:", err)
		os.Exit(1)
	}
	fmt.Println(flags)
}

func buildFlags(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	branch, err := gitValue("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if branch == "HEAD" {
		branch, err = gitValue("rev-parse", "--short", "HEAD")
		if err != nil {
			return "", err
		}
	}
	return linkerFlags(data, branch, time.Now().UTC())
}

func gitValue(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func linkerFlags(data []byte, branch string, builtAt time.Time) (string, error) {
	var config struct {
		Info struct {
			ProductName string `yaml:"productName"`
			Description string `yaml:"description"`
			Version     string `yaml:"version"`
			Copyright   string `yaml:"copyright"`
			License     string `yaml:"license"`
		} `yaml:"info"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}

	var flags []string
	for _, field := range []struct {
		source string
		symbol string
		value  string
	}{
		{"info.productName", "name", config.Info.ProductName},
		{"info.description", "tagLine", config.Info.Description},
		{"info.version", "version", config.Info.Version},
		{"info.copyright", "copyright", config.Info.Copyright},
		{"info.license", "license", config.Info.License},
		{"git branch", "branch", branch},
		{"build time", "buildDate", builtAt.UTC().Format(time.RFC3339)},
	} {
		if strings.TrimSpace(field.value) == "" || strings.ContainsAny(field.value, "\x00\r\n") {
			return "", fmt.Errorf("%s must be a non-empty, single-line value", field.source)
		}
		arg := "github.com/lmorg/ttyphoon/app." + field.symbol + "=" + field.value
		quoted, err := quoteLinkerArgument(arg)
		if err != nil {
			return "", fmt.Errorf("%s: %w", field.source, err)
		}
		flags = append(flags, "-X", quoted)
	}
	return strings.Join(flags, " "), nil
}

// Go's -ldflags parser accepts surrounding quotes but does not unescape them.
func quoteLinkerArgument(arg string) (string, error) {
	if !strings.ContainsAny(arg, " \t'\"") {
		return arg, nil
	}
	if !strings.ContainsRune(arg, '\'') {
		return "'" + arg + "'", nil
	}
	if !strings.ContainsRune(arg, '"') {
		return `"` + arg + `"`, nil
	}
	return "", fmt.Errorf("value contains both single and double quotes, which Go linker flags cannot quote")
}
