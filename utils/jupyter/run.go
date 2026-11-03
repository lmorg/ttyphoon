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
	"runtime/debug"
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

func PrepareTerminalCommand(code, language, pwd string) (string, func(), error) {
	binding := resolveFormatBinding(language)
	if binding == nil {
		return "", nil, fmt.Errorf("unsupported language: %s", language)
	}
	if len(binding.ExecuteCommand) == 0 {
		return "", nil, fmt.Errorf("no execution command for %s", binding.Description)
	}
	directory, err := os.MkdirTemp("", app.DirName+"-terminal-")
	if err != nil {
		return "", nil, errCreatingTempFile(err)
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	name := fileName("Main", binding)
	if err := writeRenderedSource(directory, name, code, binding); err != nil {
		cleanup()
		return "", nil, err
	}
	path := filepath.Join(directory, name)
	var lines, commands []string
	if pwd != "" {
		lines = append(lines, "cd -- "+terminalShellCommand([]string{pwd})+" || exit $?")
	}
	if format := expandVars(binding.FormatCommand, path); len(format) > 0 {
		lines = append(lines, terminalShellCommand(format)+" || :")
	}
	for _, argv := range [][]string{binding.BuildCommand, binding.BuildCommand2, binding.ExecuteCommand} {
		if len(argv) > 0 {
			commands = append(commands, terminalShellCommand(expandVars(argv, path)))
		}
	}
	lines = append(lines, strings.Join(commands, " && "))
	return "(\n" + strings.Join(lines, "\n") + "\n)", cleanup, nil
}

func terminalShellCommand(argv []string) string {
	quoted := make([]string, len(argv))
	for index, argument := range argv {
		quoted[index] = "'" + strings.ReplaceAll(argument, "'", "'\\''") + "'"
	}
	return strings.Join(quoted, " ")
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
			_ = execute(ctx, codeId, pwd, format, nil)
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

func logType(isErr bool) string {
	if isErr {
		return "error"
	}
	return "info"
}

func execute(ctx context.Context, codeId, pwd string, argv []string, ch chan *OutputT) int {
	if len(argv) == 0 {
		if ch != nil {
			ch <- &OutputT{
				Id:     codeId,
				Output: "I don't know how to execute this document type",
				IsErr:  true,
			}
		}
		return 1
	}

	if ch == nil {
		ch = make(chan *OutputT)
		defer close(ch)
		go func() {
			for out := range ch {
				log.Printf("[%s] %s: %s", _LOG_TYPE[out.IsErr], argv[0], out.Output)
			}
		}()
	}

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
	defer func() {
		if recover := recover(); recover != nil {
			log.Printf("[error] panic caught in jupyter/readAndEmit(): %v", recover)
			_, _ = os.Stderr.Write(debug.Stack())
		}
	}()

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
