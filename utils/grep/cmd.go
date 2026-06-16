package grep

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func buildSearchCommand(query string, opts Options) (*exec.Cmd, string, error) {
	if _, err := exec.LookPath("rg"); err == nil {
		return exec.Command("rg", buildRgArgs(query, opts)...), "rg", nil
	}

	if _, err := exec.LookPath("grep"); err == nil {
		return exec.Command("grep", buildGrepArgs(query, opts)...), "grep", nil
	}

	return nil, "", fmt.Errorf("neither ripgrep ('rg') nor grep found in PATH")
}

// buildRgArgs builds ripgrep command arguments based on options.
func buildRgArgs(query string, opts Options) []string {
	args := []string{"-uu", "-n", "--no-heading", "--color", "never"}

	if opts.Regex {
		// Regex mode (default for rg without -F)
	} else {
		// Plain text/literal mode
		args = append(args, "-F")
	}

	if !opts.CaseSensitive {
		args = append(args, "-i")
	}

	if opts.WholeWord {
		args = append(args, "-w")
	}

	// End of options marker ensures patterns beginning with '-' are treated as queries.
	args = append(args, "--", query, ".")
	return args
}

// buildGrepArgs builds grep command arguments based on options.
func buildGrepArgs(query string, opts Options) []string {
	args := []string{"-R", "-n", "--binary-files=without-match"}

	if opts.Regex {
		args = append(args, "-E")
	} else {
		// Plain text/literal mode
		args = append(args, "-F")
	}

	if !opts.CaseSensitive {
		args = append(args, "-i")
	}

	if opts.WholeWord {
		args = append(args, "-w")
	}

	// End of options marker ensures patterns beginning with '-' are treated as queries.
	args = append(args, "--", query, ".")
	return args
}

func parseOutputLine(rawLine, projectDir string) (*Result, bool) {
	result := &Result{}
	var err error

	line := strings.TrimSpace(rawLine)
	if line == "" {
		return nil, false
	}

	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 2 {
		return nil, false
	}

	result.Line, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil, false
	}

	/*result.Path = parts[0]
	if !filepath.IsAbs(result.Path) {
		result.Path = filepath.Join(projectDir, result.Path)
	}
	result.Path, _ = filepath.Abs(result.Path)*/
	result.Path = filepath.Join(projectDir, parts[0])

	return result, true
}
