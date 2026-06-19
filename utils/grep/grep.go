package grep

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	_BUF_MIN    = 64 * 1024
	_BUF_MAX    = 10 * 1024 * 1024
	_BATCH_SIZE = 50
)

var (
	cachedCmd     *exec.Cmd
	cachedSearch  string
	cachedResults []*Result
	cacheMutex    sync.Mutex
)

func readContextWithCache(path string, lineNo int, cache map[string][]string) []string {
	var lines []string

	if cached, ok := cache[path]; ok {
		lines = cached
	} else {
		lines = readLines(path)
		cache[path] = lines
	}

	idx := lineNo - 1 // convert to 0-based
	if idx < 0 || idx >= len(lines) {
		return nil
	}

	start := max(idx-1, 0)
	end := idx + 1
	if end >= len(lines) {
		end = len(lines) - 1
	}

	result := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		result = append(result, strings.TrimSpace(lines[i]))
	}

	return result
}

func readLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, _BUF_MIN), _BUF_MAX)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if scanner.Err() != nil {
		return nil
	}

	return lines
}

func BatchedStreamResults(searchRoot, query string, opts Options, pathMapper func(string) string, results chan<- []*Result) error {
	if !cacheMutex.TryLock() {
		if cachedCmd != nil && cachedCmd.Process != nil {
			_ = cachedCmd.Process.Kill()
		}
		cacheMutex.Lock()
	}

	var wg sync.WaitGroup

	defer func() {
		wg.Wait()
		close(results)
		cacheMutex.Unlock()
	}()

	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	cmd, bin, err := buildSearchCommand(query, opts)
	if err != nil {
		return err
	}

	cmd.Dir = searchRoot
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	batch := make([]*Result, 0, _BATCH_SIZE)
	contextCache := map[string][]string{}

	flushBatch := func() {
		copyBatch := make([]*Result, len(batch))
		copy(copyBatch, batch)

		go func() {
			results <- copyBatch
			wg.Done()
		}()

		batch = batch[:0]
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, _BUF_MIN), _BUF_MAX)

	var fnFilter func(*Result) bool
	filter := strings.TrimSpace(opts.FileFilter)
	if filter == "" {
		fnFilter = func(m *Result) bool { return true }
	} else {
		filter = strings.ToLower(filter)
		fnFilter = func(m *Result) bool {
			return strings.Contains(strings.ToLower(m.Path), filter)
		}
	}

	wg.Add(1)
	for scanner.Scan() {
		match, ok := parseOutputLine(scanner.Text(), searchRoot)
		if !ok {
			continue
		}

		if !fnFilter(match) {
			continue
		}

		match.Context = readContextWithCache(match.Path, match.Line, contextCache)
		match.FileName = filepath.Base(match.Path)
		match.Path = pathMapper(match.Path)

		batch = append(batch, match)

		if len(batch) >= _BATCH_SIZE {
			flushBatch()
			wg.Add(1)
		}
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Process.Kill()
		return err
	}

	if len(batch) > 0 {
		flushBatch()
	} else {
		wg.Done()
	}

	msg := strings.TrimSpace(stderr.String())
	if msg != "" {
		return fmt.Errorf("%s failed: %s", bin, msg)
	}

	return nil
}
