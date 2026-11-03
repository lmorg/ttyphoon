package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type gitInfoTestRenderer struct {
	types.Renderer
	events []string
}

type gitInfoTestNotification struct {
	types.Notification
	renderer *gitInfoTestRenderer
}

func (notification *gitInfoTestNotification) Close() {
	notification.renderer.events = append(notification.renderer.events, "close")
}

func (renderer *gitInfoTestRenderer) DisplaySticky(level types.NotificationType, message string, cancel func()) types.Notification {
	renderer.events = append(renderer.events, level.String()+":"+message)
	return &gitInfoTestNotification{renderer: renderer}
}

func (renderer *gitInfoTestRenderer) DisplayMarkdownModel(id, markdown string) {
	renderer.events = append(renderer.events, "model:"+id+":"+markdown)
}

func TestGitInfoStickyClosesBeforeModelAndOnError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		renderer := &gitInfoTestRenderer{}
		generationError := errors.New("generation failed")
		err := showGitInfoModel(renderer, "git-info:test", func() (string, error) {
			renderer.events = append(renderer.events, "generate")
			if fail {
				return "", generationError
			}
			return "# Info", nil
		})
		want := []string{"info:Generating diffs", "generate", "close"}
		if fail {
			if !errors.Is(err, generationError) {
				t.Fatalf("generation error = %v", err)
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			want = append(want, "model:git-info:test:# Info")
		}
		if !reflect.DeepEqual(renderer.events, want) {
			t.Fatalf("events = %v, want %v", renderer.events, want)
		}
	}
}

func TestGitStatusCacheReusesAndRefreshesSnapshots(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.invalid"},
	} {
		command := exec.Command(gitPath, append([]string{"-C", root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	filePath := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(filePath, []byte("head\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "--", "tracked.txt"}, {"commit", "--quiet", "-m", "initial"}} {
		command := exec.Command(gitPath, append([]string{"-C", root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}

	app := &WApp{}
	initial, err := app.GetGitStatus(root)
	if err != nil {
		t.Fatalf("GetGitStatus() error = %v", err)
	}
	if !initial.Clean {
		t.Fatalf("initial status = %+v, want clean", initial)
	}

	if err := os.WriteFile(filePath, []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cached, err := app.GetGitStatus(root)
	if err != nil || !cached.Clean {
		t.Fatalf("cached status = %+v, %v; want original clean snapshot", cached, err)
	}

	refreshed, err := app.RefreshGitStatus(root)
	if err != nil {
		t.Fatalf("RefreshGitStatus() error = %v", err)
	}
	if refreshed.Clean || len(refreshed.Files) != 1 || !refreshed.Files[0].Unstaged {
		t.Fatalf("refreshed status = %+v, want one unstaged file", refreshed)
	}
	cached, err = app.GetGitStatus(root)
	if err != nil || cached.Clean || len(cached.Files) != 1 {
		t.Fatalf("GetGitStatus() after refresh = %+v, %v", cached, err)
	}
}
