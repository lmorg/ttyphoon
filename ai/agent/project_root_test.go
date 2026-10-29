package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type projectRootTestTile struct {
	types.Tile
	pwd string
}

func (tile *projectRootTestTile) Pwd() string { return tile.pwd }

type projectRootTestTerm struct {
	types.Term
	tile types.Tile
}

func (term *projectRootTestTerm) Tile() types.Tile { return term.tile }

type projectRootTestRenderer struct {
	types.Renderer
	active types.Tile
}

func (renderer *projectRootTestRenderer) ActiveTile() types.Tile { return renderer.active }

func TestProjectRootTracksOwningPaneDirectoryRatherThanFocusedPane(t *testing.T) {
	firstRoot, secondRoot := t.TempDir(), t.TempDir()
	for _, root := range []string{firstRoot, secondRoot} {
		if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	pane := &projectRootTestTile{pwd: filepath.Join(firstRoot, "nested")}
	renderer := &projectRootTestRenderer{active: &projectRootTestTile{pwd: secondRoot}}
	agt := &Agent{term: &projectRootTestTerm{tile: pane}, renderer: renderer}
	if got := agt.ProjectRoot(); got != firstRoot {
		t.Fatalf("initial project root = %q, want owning pane root %q", got, firstRoot)
	}
	pane.pwd = filepath.Join(secondRoot, "nested")
	renderer.active = &projectRootTestTile{pwd: firstRoot}
	if got := agt.ProjectRoot(); got != secondRoot {
		t.Fatalf("project root after directory change = %q, want %q", got, secondRoot)
	}
}

func TestProjectRootDoesNotUseApplicationWorkingDirectoryWhenPanePathIsEmpty(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	applicationRepo := t.TempDir()
	if err := os.Mkdir(filepath.Join(applicationRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(applicationRepo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	agt := &Agent{term: &projectRootTestTerm{tile: &projectRootTestTile{pwd: ""}}}
	if got := agt.ProjectRoot(); got != "" {
		t.Fatalf("empty pane PWD resolved to application directory %q", got)
	}
}
