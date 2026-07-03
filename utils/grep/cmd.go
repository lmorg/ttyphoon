package grep

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lmorg/ttyphoon/config"
)

func buildSearchCommand(query string, opts Options) (*exec.Cmd, string, error) {
	if _, err := exec.LookPath("rg"); err == nil {
		log.Println(`[debug] grep: cmd="rg"`)
		return exec.Command("rg", buildRgArgs(query, opts)...), "rg", nil
	}

	if _, err := exec.LookPath("grep"); err == nil {
		log.Println(`[debug] grep: cmd="grep"`)
		return exec.Command("grep", buildGrepArgs(query, opts)...), "grep", nil
	}

	return nil, "", fmt.Errorf("neither ripgrep ('rg') nor grep found in PATH")
}

// buildRgArgs builds ripgrep command arguments based on options.
func buildRgArgs(query string, opts Options) []string {
	// -uu makes ripgrep search hidden and .gitignore'd files, mirroring grep's
	// "search everything" recursion. The glob excludes then restore the same
	// directory exclusions that buildGrepArgs applies via --exclude-dir, so both
	// backends skip .git and node_modules consistently. --follow follows
	// symlinks to match grep's -R behaviour.
	args := []string{
		"-uu", "-n", "--no-heading", "--color", "never", "--follow",
		//"-g", "!.git", "-g", "!node_modules",
	}

	for dir, ignore := range config.Config.Notes.ExcludeDirectories {
		if ignore {
			args = append(args, "-g", "!"+dir)
		}
	}

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
	args := []string{
		"-R", "-n", "--binary-files=without-match",
		//"--exclude-dir=.git", "--exclude-dir=node_modules",
	}

	for dir, ignore := range config.Config.Notes.ExcludeDirectories {
		if ignore {
			args = append(args, "--exclude-dir="+dir)
		}
	}

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
