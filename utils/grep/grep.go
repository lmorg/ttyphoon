package grep

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Result represents a single search result with JSON tags for Wails binding.
type Result struct {
	FileName string   `json:"fileName"`
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Context  []string `json:"context"`
}

// ReturnValue represents the return value for grep searches with results and error.
type ReturnValue struct {
	Results []Result `json:"results"`
	Error   string   `json:"error"`
}

// Match represents a single search hit in a file.
type Match struct {
	FileName string
	Path     string
	Line     int
	// Context holds up to 3 lines: [line before match, matching line, line after match].
	// Absent lines (e.g. at start/end of file) are omitted; the slice may be 1–3 items long.
	Context []string
}

// Options configures the search behavior.
type Options struct {
	CaseSensitive bool
	Regex         bool
	WholeWord     bool
}

// SearchProject searches from the current working directory.
func SearchProject(query string) ([]Match, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return Search(cwd, query)
}

// SearchProjectWithOptions searches from the current working directory with options.
func SearchProjectWithOptions(query string, opts Options) ([]Match, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return SearchWithOptions(cwd, query, opts)
}

// Search looks for query in files under projectDir.
// It prefers ripgrep and falls back to grep.
func Search(projectDir, query string) ([]Match, error) {
	projectDir = strings.TrimSpace(projectDir)
	if projectDir == "" {
		return nil, fmt.Errorf("project directory cannot be empty")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	return SearchWithOptions(projectDir, query, Options{})
}

// SearchWithOptions looks for query in files under projectDir with given options.
// It prefers ripgrep and falls back to grep.
func SearchWithOptions(projectDir, query string, opts Options) ([]Match, error) {
	projectDir = strings.TrimSpace(projectDir)
	if projectDir == "" {
		return nil, fmt.Errorf("project directory cannot be empty")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if _, err := exec.LookPath("rg"); err == nil {
		return runSearch("rg", buildRgArgs(query, opts), projectDir)
	}

	if _, err := exec.LookPath("grep"); err == nil {
		return runSearch("grep", buildGrepArgs(query, opts), projectDir)
	}

	return nil, fmt.Errorf("neither ripgrep ('rg') nor grep found in PATH")
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

	args = append(args, query, ".")
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

	args = append(args, query, ".")
	return args
}

func runSearch(bin string, args []string, dir string) ([]Match, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means no matches for both rg and grep.
			return []Match{}, nil
		}

		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%s failed: %s", bin, msg)
		}
		return nil, fmt.Errorf("%s failed: %w", bin, err)
	}

	return parseOutput(stdout.String(), dir)
}

func parseOutput(output, projectDir string) ([]Match, error) {
	matches := make([]Match, 0)
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 2 {
			continue
		}

		lineNo, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		absPath := parts[0]
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(projectDir, absPath)
		}
		absPath, _ = filepath.Abs(absPath)

		matches = append(matches, Match{
			FileName: filepath.Base(absPath),
			Path:     absPath,
			Line:     lineNo,
			Context:  readContext(absPath, lineNo),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

// readContext returns up to 3 lines around lineNo (1-based):
// the line before, the matching line, and the line after.
// Missing lines at file boundaries are simply omitted.
func readContext(path string, lineNo int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var all []string
	for scanner.Scan() {
		all = append(all, scanner.Text())
	}
	if scanner.Err() != nil || len(all) == 0 {
		return nil
	}

	idx := lineNo - 1 // convert to 0-based
	if idx < 0 || idx >= len(all) {
		return nil
	}

	start := idx - 1
	if start < 0 {
		start = 0
	}
	end := idx + 1
	if end >= len(all) {
		end = len(all) - 1
	}

	result := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		result = append(result, all[i])
	}
	return result
}

// SearchAndReturn performs a search and returns results with error handling for API consumption.
// It handles empty queries and returns a ReturnValue suitable for Wails bindings.
// Optional pathMapper function can transform file paths (e.g., for history mapping).
func SearchAndReturn(searchRoot, query string, opts Options, pathMapper func(string) string) ReturnValue {
	query = strings.TrimSpace(query)
	if query == "" {
		return ReturnValue{Results: []Result{}}
	}

	searchRoot = strings.TrimSpace(searchRoot)
	if searchRoot == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ReturnValue{Error: err.Error()}
		}
		searchRoot = wd
	}

	matches, err := SearchWithOptions(searchRoot, query, opts)
	if err != nil {
		return ReturnValue{Error: err.Error()}
	}

	results := make([]Result, 0, len(matches))
	for i := range matches {
		path := matches[i].Path
		if pathMapper != nil {
			if mapped := pathMapper(path); mapped != "" {
				path = mapped
			}
		}

		fileName := matches[i].FileName
		if fileName == "" {
			fileName = filepath.Base(path)
		}

		results = append(results, Result{
			FileName: fileName,
			Path:     path,
			Line:     matches[i].Line,
			Context:  matches[i].Context,
		})
	}

	return ReturnValue{Results: results}
}
