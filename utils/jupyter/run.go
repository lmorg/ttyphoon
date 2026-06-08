package jupyter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

type cacheT struct {
	code string
	mu   sync.Mutex
}

var (
	cache      = map[string]*cacheT{}
	cacheMutex sync.Mutex
)

func tempDir(bookId string) string {
	return filepath.Join(os.TempDir(), app.DirName, "runbook", bookId)
}

func fileName(codeId string, binding *LanguageBindingT) string {
	return fmt.Sprintf("%s.%s", codeId, binding.FileExtension)
}

func runNote(ctx context.Context, bookId, codeId, pwd, code string, ch chan *OutputT, binding *LanguageBindingT, parameters ...string) {
	tempDir := tempDir(bookId)
	fileName := fileName(codeId, binding)
	filePath := filepath.Join(tempDir, fileName)

	var exitCode int

	cacheMutex.Lock()
	c := cache[filePath]
	cached := true
	if c == nil {
		cached = false
		c = &cacheT{code: code}
		cache[filePath] = c
	}
	c.mu.Lock()
	cacheMutex.Unlock()

	cached = cached && c.code == code

	if !cached {
		err := writeRenderedSource(tempDir, fileName, code, binding)
		if err != nil {
			ch <- &OutputT{
				Id:     codeId,
				Output: err.Error(),
				IsErr:  true,
			}
			c.mu.Unlock()
			return
		}

		format := expandVars(binding.FormatCommand, filePath)
		if len(format) > 0 {
			execute(ctx, codeId, pwd, format, ch)
		}

		build1 := expandVars(binding.BuildCommand, filePath)
		if len(build1) > 0 && exitCode == 0 {
			exitCode = execute(ctx, codeId, pwd, build1, ch)
		}
		build2 := expandVars(binding.BuildCommand2, filePath)
		if len(build2) > 0 && exitCode == 0 {
			exitCode = execute(ctx, codeId, pwd, build2, ch)
		}
	}

	c.mu.Unlock()

	if exitCode == 0 {
		var exe []string
		if len(parameters) > 0 {
			exe = expandParameters(binding.ExecuteParameters, filePath, parameters)
		} else {
			exe = expandVars(binding.ExecuteCommand, filePath)
		}
		_ = execute(ctx, codeId, pwd, exe, ch)
	}

	go func() {
		time.Sleep(250 * time.Millisecond) // just to avoid any chance of the channel closing before the output has finished being flushed
		close(ch)
	}()
}

func execute(ctx context.Context, codeId, pwd string, argv []string, ch chan *OutputT) int {
	select {
	case <-ctx.Done():
		return 1
	default:
	}

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)

	cmd.Dir = pwd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		ch <- &OutputT{
			Id:     codeId,
			Output: fmt.Sprintf("Error getting stdout: %v", err),
			IsErr:  true,
		}
		return 1
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		ch <- &OutputT{
			Id:     codeId,
			Output: fmt.Sprintf("Error getting stderr: %v", err),
			IsErr:  true,
		}
		return 1
	}

	err = cmd.Start()
	if err != nil {
		ch <- &OutputT{
			Id:     codeId,
			Output: fmt.Sprintf("Error starting command: %v", err),
			IsErr:  true,
		}
		return 1
	}

	go readAndEmit(codeId, ch, stdout, false)
	go readAndEmit(codeId, ch, stderr, true)

	err = cmd.Wait()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			ch <- &OutputT{
				Id:     codeId,
				Output: fmt.Sprintf("Error starting command: %v", err),
				IsErr:  true,
			}
		}
	}

	return cmd.ProcessState.ExitCode()
}

var _LOG_TYPE = map[bool]string{true: "warn", false: "info"}

func readAndEmit(id string, ch chan *OutputT, reader io.Reader, isStderr bool) {
	if ch == nil {
		ch = make(chan *OutputT)

		go func() {
			for output := range ch {
				log.Printf("[%s] Code formatter: %s", _LOG_TYPE[output.IsErr], output.Output)
			}
		}()
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		ch <- &OutputT{
			Id:     id,
			Output: line,
			IsErr:  isStderr,
		}
	}

	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "file already closed") {
			return
		}

		ch <- &OutputT{
			Id:     id,
			Output: fmt.Sprintf("Error reading output: %v", err),
			IsErr:  true,
		}
	}
}
