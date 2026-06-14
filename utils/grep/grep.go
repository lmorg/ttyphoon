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

// Match represents a single search hit in a file.
type Match struct {
	FileName string
	Path     string
	Line     int
	// Context holds up to 3 lines: [line before match, matching line, line after match].
	// Absent lines (e.g. at start/end of file) are omitted; the slice may be 1–3 items long.
	Context []string
}

// SearchProject searches from the current working directory.
func SearchProject(query string) ([]Match, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return Search(cwd, query)
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

	if _, err := exec.LookPath("rg"); err == nil {
		return runSearch("rg", []string{"-uu", "-n", "-F", "--no-heading", "--color", "never", query, "."}, projectDir)
	}

	if _, err := exec.LookPath("grep"); err == nil {
		return runSearch("grep", []string{"-R", "-n", "-F", "--binary-files=without-match", query, "."}, projectDir)
	}

	return nil, fmt.Errorf("neither ripgrep ('rg') nor grep found in PATH")
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
