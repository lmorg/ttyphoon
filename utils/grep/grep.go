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

	"github.com/lmorg/ttyphoon/utils/find"
)

const (
	_BUF_MIN    = 64 * 1024
	_BUF_MAX    = 10 * 1024 * 1024
	_BATCH_SIZE = 50
)

var (
	cachedRoot    string
	cachedQuery   string
	cachedResults []*Result
	cacheMutex    sync.Mutex

	// cachedCmd holds the currently running search command so an incoming
	// search can interrupt it. It has its own mutex because it must be read and
	// killed while cacheMutex is still held by the in-flight search.
	cachedCmd *exec.Cmd
	cmdMutex  sync.Mutex
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
		// Another search is already running. Interrupt it by killing its process
		// so it unwinds quickly, then wait for the lock. cachedCmd is guarded by
		// cmdMutex so it can be accessed while the in-flight search still holds
		// cacheMutex. The interrupted search invalidates its own cache entry, so
		// we must not touch cachedRoot/cachedQuery here.
		cmdMutex.Lock()
		if cachedCmd != nil && cachedCmd.Process != nil {
			_ = cachedCmd.Process.Kill()
		}
		cmdMutex.Unlock()
		cacheMutex.Lock()
	}

	query = strings.TrimSpace(query)
	if query == "" {
		close(results)
		cacheMutex.Unlock()
		return nil
	}

	if cachedRoot == searchRoot && cachedQuery == query {
		var chunk []*Result
		fnFilter := filterFunc(opts)
		for _, r := range cachedResults {
			if fnFilter(r) {
				chunk = append(chunk, r)
			}
			//if len(chunk) == 50 {
			//	results <- chunk
			//	chunk = []*Result{}
			//}
		}
		results <- chunk
		close(results)
		cacheMutex.Unlock()
		return nil
	}

	return batchedStreamResults(searchRoot, query, opts, pathMapper, results)
}

func batchedStreamResults(searchRoot, query string, opts Options, pathMapper func(string) string, results chan<- []*Result) error {
	cachedRoot = searchRoot
	cachedQuery = query
	cachedResults = []*Result{}

	var wg sync.WaitGroup

	defer func() {
		wg.Wait()
		cmdMutex.Lock()
		cachedCmd = nil
		cmdMutex.Unlock()
		close(results)
		cacheMutex.Unlock()
	}()

	cmd, bin, err := buildSearchCommand(query, opts)
	if err != nil {
		return err
	}

	cmd.Dir = searchRoot

	cmdMutex.Lock()
	cachedCmd = cmd
	cmdMutex.Unlock()

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
		copyBatch := make([]*Result, len(batch))
		copy(copyBatch, batch)

		// Cache the full, unfiltered batch so the cache is independent of the
		// active FileFilter. A later search reusing this (root, query) with a
		// broader filter can then recover the previously-excluded results from the
		// cache. Only the filtered subset is streamed to the consumer.
		cachedResults = append(cachedResults, copyBatch...)

		filtered := make([]*Result, 0, len(copyBatch))
		for _, r := range copyBatch {
			if fnFilter(r) {
				filtered = append(filtered, r)
			}
		}

		go func() {
			results <- filtered
			wg.Done()
		}()

		batch = batch[:0]
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, _BUF_MIN), _BUF_MAX)

	wg.Add(1)
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
			wg.Add(1)
		}
	}

	if err := scanner.Err(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait() // reap the killed process
		// The read failed mid-stream, so the cached results are incomplete: drop
		// them. Also balance the outstanding wg.Add(1) for the in-progress batch
		// so the deferred wg.Wait() can return instead of deadlocking.
		invalidateCache()
		wg.Done()
		return err
	}

	if len(batch) > 0 {
		flushBatch()
	} else {
		wg.Done()
	}

	// Reap the child to avoid leaving a zombie process, and to synchronise on
	// the stderr copier goroutine so stderr is fully populated before we read
	// it. A non-zero exit (e.g. grep/rg returning 1 for "no matches") is not an
	// error for us; failure is determined from stderr below.
	waitErr := cmd.Wait()

	msg := strings.TrimSpace(stderr.String())
	if msg != "" {
		// Command failed: the cached results are unreliable, so drop them.
		invalidateCache()
		return fmt.Errorf("%s failed: %s", bin, msg)
	}

	// If the process was killed (exit code -1 means terminated by a signal), a
	// newer search interrupted this one. The result set is incomplete, so drop
	// it rather than serving a partial result for this query later.
	if exitErr, ok := waitErr.(*exec.ExitError); ok && exitErr.ProcessState.ExitCode() == -1 {
		invalidateCache()
	}

	return nil
}

// invalidateCache clears the cached result set. Callers must hold cacheMutex.
func invalidateCache() {
	cachedRoot = ""
	cachedQuery = ""
	cachedResults = nil
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
