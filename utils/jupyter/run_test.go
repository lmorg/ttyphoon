package jupyter

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareTerminalCommandUsesBindingAndRunsStagesInProject(t *testing.T) {
	tool := filepath.Join(t.TempDir(), "synthetic tool's runner")
	script := `#!/bin/sh
printf '%s\n' "$1" >> "$RUN_LOG"
case "$1" in
    format) exit 9 ;;
    build) exit "$BUILD_EXIT" ;;
    link) exit "$LINK_EXIT" ;;
    execute) printf '%s\n%s\n' "$PWD" "$3" ;;
    *) exit 99 ;;
esac
`
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	literal := "literal ';$(not-a-command)"
	binding := &LanguageBindingT{
		Aliases: []string{"fixture"}, Description: "Synthetic runtime",
		Template: "before\n{{ .Code }}\nafter", FileExtension: "fixture",
		FormatCommand:  []string{tool, "format", "${FILE}"},
		BuildCommand:   []string{tool, "build", "${FILE}"},
		BuildCommand2:  []string{tool, "link", "${FILE}"},
		ExecuteCommand: []string{tool, "execute", "${FILE}", literal},
	}
	original := Languages
	Languages = []*LanguageBindingT{binding}
	t.Cleanup(func() { Languages = original })
	for _, test := range []struct {
		name, language, buildExit, linkExit, stages string
		exitCode                                    int
	}{
		{"success", binding.Description, "0", "0", "format\nbuild\nlink\nexecute\n", 0},
		{"build failure", "fixture", "7", "0", "format\nbuild\n", 7},
		{"second build failure", "fixture", "0", "8", "format\nbuild\nlink\n", 8},
	} {
		t.Run(test.name, func(t *testing.T) {
			temp := t.TempDir()
			t.Setenv("TMPDIR", temp)
			project := filepath.Join(t.TempDir(), "project's working dir")
			if err := os.Mkdir(project, 0700); err != nil {
				t.Fatal(err)
			}
			project, err := filepath.EvalSymlinks(project)
			if err != nil {
				t.Fatal(err)
			}
			command, cleanup, err := PrepareTerminalCommand("example code", test.language, project)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(cleanup)
			files, err := filepath.Glob(filepath.Join(temp, "*", "Main.fixture"))
			if err != nil || len(files) != 1 {
				t.Fatalf("source files = %v, %v", files, err)
			}
			source, err := os.ReadFile(files[0])
			if err != nil || string(source) != "before\nexample code\nafter" {
				t.Fatalf("rendered source = %q, %v", source, err)
			}
			log := filepath.Join(t.TempDir(), "stages")
			if _, err := os.Stat(log); !os.IsNotExist(err) {
				t.Fatal("preparing the source executed a command")
			}
			process := exec.Command("/bin/sh", "-c", command)
			process.Dir = t.TempDir()
			process.Env = append(os.Environ(), "RUN_LOG="+log, "BUILD_EXIT="+test.buildExit, "LINK_EXIT="+test.linkExit)
			output, runErr := process.CombinedOutput()
			if process.ProcessState == nil || process.ProcessState.ExitCode() != test.exitCode {
				t.Fatalf("pipeline exit = %v, output = %s", runErr, output)
			}
			stages, err := os.ReadFile(log)
			if err != nil || string(stages) != test.stages {
				t.Fatalf("pipeline stages = %q, %v; want %q", stages, err, test.stages)
			}
			if test.exitCode == 0 && string(output) != project+"\n"+literal+"\n" {
				t.Fatalf("working directory or literal arguments changed: %q", output)
			}
			cleanup()
			if _, err := os.Stat(filepath.Dir(files[0])); !os.IsNotExist(err) {
				t.Fatalf("source artifacts remain after cleanup: %v", err)
			}
		})
	}
}

func TestPrepareTerminalCommandRejectsUnsupportedRuntime(t *testing.T) {
	command, cleanup, err := PrepareTerminalCommand("code", "unsupported synthetic runtime", "")
	if err == nil || command != "" || cleanup != nil {
		t.Fatalf("unsupported runtime = %q, %v", command, err)
	}
}

func TestPrepareTerminalCommandUsesDistinctFiles(t *testing.T) {
	first, cleanupFirst, err := PrepareTerminalCommand("echo first", "bash", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupFirst()
	second, cleanupSecond, err := PrepareTerminalCommand("echo second", "bash", "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupSecond()
	if first == second || !strings.Contains(first, "Main.sh") || !strings.Contains(second, "Main.sh") {
		t.Fatal("terminal invocations did not get distinct source paths")
	}
}
