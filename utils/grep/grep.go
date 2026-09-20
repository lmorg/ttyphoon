package grep

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/utils/find"
)

const (
	_BUF_MIN    = 64 * 1024
	_BUF_MAX    = 10 * 1024 * 1024
	_BATCH_SIZE = 50
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

func BatchedStreamResults(ctx context.Context, searchRoot, query string, opts Options, pathMapper func(string) string, results chan<- []*Result) error {
	defer close(results)
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	return batchedStreamResults(ctx, searchRoot, query, opts, pathMapper, results)
}

func batchedStreamResults(ctx context.Context, searchRoot, query string, opts Options, pathMapper func(string) string, results chan<- []*Result) error {
	cmd, bin, err := buildSearchCommand(ctx, query, opts)
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

	fnFilter := filterFunc(opts)
	batch := make([]*Result, 0, _BATCH_SIZE)
	contextCache := map[string][]string{}

	flushBatch := func() {
		filtered := make([]*Result, 0, len(batch))
		for _, r := range batch {
			if fnFilter(r) {
				filtered = append(filtered, r)
			}
		}
		select {
		case results <- filtered:
		case <-ctx.Done():
		}

		batch = batch[:0]
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, _BUF_MIN), _BUF_MAX)

	for scanner.Scan() {
		match, ok := parseOutputLine(scanner.Text(), searchRoot)
		if !ok {
			continue
		}

		match.Context = readContextWithCache(match.Path, match.Line, contextCache)
		match.FileName = filepath.Base(match.Path)
		match.Path = pathMapper(match.Path)

		batch = append(batch, match)

		if len(batch) >= _BATCH_SIZE {
			flushBatch()
		}
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return err
	}

	if len(batch) > 0 {
		flushBatch()
	}

	// Reap the child to avoid leaving a zombie process, and to synchronise on
	// the stderr copier goroutine so stderr is fully populated before we read
	// it. A non-zero exit (e.g. grep/rg returning 1 for "no matches") is not an
	// error for us; failure is determined from stderr below.
	_ = cmd.Wait()

	msg := strings.TrimSpace(stderr.String())
	if msg != "" {
		return fmt.Errorf("%s failed: %s", bin, msg)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

func filterFunc(opts Options) func(*Result) bool {
	filter := strings.TrimSpace(opts.FileFilter)
	if filter == "" {
		return func(m *Result) bool { return true }
	} else {
		// Use the find package so the file filter supports the same rich
		// syntax as the file-list filter: plain words (AND), "or" (OR),
		// "!" (NOT), "rx" (regexp) and "g" (glob). Fall back to a plain
		// case-insensitive substring match if the pattern is malformed.
		f, err := find.New(filter)
		if err != nil {
			//lower := strings.ToLower(filter)
			//return func(m *Result) bool {
			//	return strings.Contains(strings.ToLower(m.Path), lower)
			//}
			return func(m *Result) bool { return true }
		} else {
			return func(m *Result) bool { return f.MatchString(m.Path) }
		}
	}
}
