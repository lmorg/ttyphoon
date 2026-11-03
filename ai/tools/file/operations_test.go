package filetools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

func TestFilesystemManagementToolsDefaultEnabledAndRequireApproval(t *testing.T) {
	root := t.TempDir()
	agt := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	tools := []aitypes.Tool{&CreateDirectory{}, &DeleteDirectory{}, &DeleteFile{}, &RenamePath{}}
	for _, prototype := range tools {
		tool, err := prototype.New(agt)
		if err != nil {
			t.Fatalf("%s.New(): %v", prototype.Name(), err)
		}
		if !tool.Enabled() {
			t.Errorf("%s is disabled by default", tool.Name())
		}
		permissions, ok := tool.(interface {
			DefaultPermissions() aitypes.DefaultPermissions
		})
		if !ok || permissions.DefaultPermissions().Invocation != "askPermission" {
			t.Errorf("%s does not require approval by default", tool.Name())
		}
	}
}

func TestCreateDirectoryCreatesNestedPathAndDescribesIt(t *testing.T) {
	root := t.TempDir()
	agt := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	tool := &CreateDirectory{agent: agt, enabled: true}
	request := &createDirectoryInputT{Path: "nested/new"}
	description := tool.CallDescription(context.Background(), request)
	if !strings.Contains(description, "nested/new") || !strings.Contains(description, "will be created") {
		t.Fatalf("authorization description = %q", description)
	}
	result, err := tool.create(request.Path)
	if err != nil || strings.Contains(result, "ERROR") {
		t.Fatalf("create directory = %q, %v", result, err)
	}
	if info, err := os.Stat(filepath.Join(root, "nested", "new")); err != nil || !info.IsDir() {
		t.Fatalf("created directory status = %v, %v", info, err)
	}
}

func TestDeleteFileRemovesOnlyRegularFiles(t *testing.T) {
	root := t.TempDir()
	agt := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	filename := filepath.Join(root, "delete-me.txt")
	if err := os.WriteFile(filename, []byte("contents"), 0600); err != nil {
		t.Fatal(err)
	}
	tool := &DeleteFile{agent: agt, enabled: true}
	result, err := tool.remove("delete-me.txt")
	if err != nil || strings.Contains(result, "ERROR") {
		t.Fatalf("delete file = %q, %v", result, err)
	}
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		t.Fatalf("file still exists or stat failed unexpectedly: %v", err)
	}
	result, err = tool.remove(".")
	if err != nil || !strings.Contains(result, "non-regular file") {
		t.Fatalf("directory delete-file attempt = %q, %v", result, err)
	}
}

func TestDeleteDirectoryRemovesEmptyOnlyAndProtectsProjectRoot(t *testing.T) {
	root := t.TempDir()
	agt := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	tool := &DeleteDirectory{agent: agt, enabled: true}
	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0755); err != nil {
		t.Fatal(err)
	}
	description := tool.CallDescription(context.Background(), &deleteDirectoryInputT{Path: "empty"})
	if !strings.Contains(description, "must be empty") {
		t.Fatalf("authorization description = %q", description)
	}
	if result, err := tool.remove("empty"); err != nil || strings.Contains(result, "ERROR") {
		t.Fatalf("delete empty directory = %q, %v", result, err)
	}
	if _, err := os.Stat(empty); !os.IsNotExist(err) {
		t.Fatalf("empty directory remains or stat failed: %v", err)
	}

	nonEmpty := filepath.Join(root, "non-empty")
	if err := os.Mkdir(nonEmpty, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmpty, "keep.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if result, err := tool.remove("non-empty"); err != nil || !strings.Contains(result, "only empty directories") {
		t.Fatalf("delete non-empty directory = %q, %v", result, err)
	}
	if result, err := tool.remove(root); err != nil || !strings.Contains(result, "project root") {
		t.Fatalf("delete project root = %q, %v", result, err)
	}
}

func TestRenamePathMovesFilesAndDirectoriesAndCreatesDestinationParents(t *testing.T) {
	root := t.TempDir()
	agt := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	tool := &RenamePath{agent: agt, enabled: true}
	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("file data"), 0600); err != nil {
		t.Fatal(err)
	}
	description := tool.CallDescription(context.Background(), &renamePathInputT{Source: "file.txt", Destination: "new/nested/file.txt"})
	if !strings.Contains(description, "new/nested") || !strings.Contains(description, "will be created") {
		t.Fatalf("rename authorization description = %q", description)
	}
	result, err := tool.rename("file.txt", "new/nested/file.txt")
	if err != nil || strings.Contains(result, "ERROR") {
		t.Fatalf("rename file = %q, %v", result, err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatalf("source file remains after rename: %v", err)
	}
	if contents, err := os.ReadFile(filepath.Join(root, "new", "nested", "file.txt")); err != nil || string(contents) != "file data" {
		t.Fatalf("renamed file contents = %q, %v", contents, err)
	}

	directory := filepath.Join(root, "tree")
	if err := os.Mkdir(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "keep.txt"), []byte("tree data"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err = tool.rename("tree", "moved/tree")
	if err != nil || strings.Contains(result, "ERROR") {
		t.Fatalf("rename directory = %q, %v", result, err)
	}
	if contents, err := os.ReadFile(filepath.Join(root, "moved", "tree", "keep.txt")); err != nil || string(contents) != "tree data" {
		t.Fatalf("renamed directory contents = %q, %v", contents, err)
	}
}

func TestRenamePathRefusesExistingDestinationSymlinksProjectRootAndSelfMove(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	agt := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	tool := &RenamePath{agent: agt, enabled: true}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.txt"), []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	if result, err := tool.rename("source.txt", "target.txt"); err != nil || !strings.Contains(result, "destination already exists") {
		t.Fatalf("existing destination rename = %q, %v", result, err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside-link")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if result, err := tool.rename("source.txt", "outside-link/new.txt"); err != nil || !strings.Contains(result, "outside the project root") {
		t.Fatalf("outside symlink destination rename = %q, %v", result, err)
	}
	if result, err := tool.rename(root, "renamed-root"); err != nil || !strings.Contains(result, "project root") {
		t.Fatalf("project root rename = %q, %v", result, err)
	}
	if err := os.Mkdir(filepath.Join(root, "tree"), 0755); err != nil {
		t.Fatal(err)
	}
	if result, err := tool.rename("tree", "tree/child/new-tree"); err != nil || !strings.Contains(result, "into itself") {
		t.Fatalf("directory self-move = %q, %v", result, err)
	}
}
