package notes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestScanFilesCancelledDoesNotPublishFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := ScanFiles(ctx, FileContext{PathProjectRoot: t.TempDir()})
	if !errors.Is(result.Err, context.Canceled) || result.Files != nil {
		t.Fatalf("cancelled scan = %+v, want cancellation with no files", result)
	}
}

func TestScanFilesUsesCapturedRoots(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "example.md"), []byte("example"), 0600); err != nil {
		t.Fatal(err)
	}
	result := ScanFiles(context.Background(), FileContext{PathProjectRoot: root})
	if result.Err != nil || len(result.Files) != 1 || result.Files[0] != "$PROJECT/example.md" {
		t.Fatalf("scan = %+v, want captured root's example.md", result)
	}
}

type cancelOnWalkContext struct {
	context.Context
	cancel context.CancelFunc
}

func (ctx cancelOnWalkContext) Done() <-chan struct{} {
	ctx.cancel()
	return ctx.Context.Done()
}

func TestScanFilesCancellationDuringWalkDiscardsAllSections(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "example.md"), []byte("example"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := ScanFiles(cancelOnWalkContext{Context: ctx, cancel: cancel}, FileContext{
		PathProjectRoot: root,
		PathGlobalNotes: root + string(filepath.Separator),
	})
	if !errors.Is(result.Err, context.Canceled) || result.Files != nil {
		t.Fatalf("cancelled traversal = %+v, want no partial sections", result)
	}
}

type unknownCwdTile struct{ types.Tile }

func (unknownCwdTile) Pwd() string       { return "" }
func (unknownCwdTile) GroupName() string { return "Unknown" }

func TestResolveFileContextDoesNotFallbackForUnknownPaneCwd(t *testing.T) {
	roots := ResolveFileContext(unknownCwdTile{})
	if roots.PathProjectRoot != "" || roots.GroupName != "Unknown" {
		t.Fatalf("unknown pane context = %+v, want no project root", roots)
	}
}
