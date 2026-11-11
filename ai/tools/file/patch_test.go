package filetools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

type pathTestAgent struct {
	projectRoot string
}

func (a *pathTestAgent) Renderer() types.Renderer                             { return nil }
func (a *pathTestAgent) ServiceName() string                                  { return "test" }
func (a *pathTestAgent) GetMeta() *aitypes.Meta                               { return nil }
func (a *pathTestAgent) ProjectRoot() string                                  { return a.projectRoot }
func (a *pathTestAgent) RequestWritePermission(context.Context, string) error { return nil }
func (a *pathTestAgent) EnvironmentValue(string) string                       { return "" }
func (a *pathTestAgent) ImageGenerationEnvironmentValue(string) string        { return "" }

type dynamicPathTestAgent struct {
	directoryTestAgent
	meta aitypes.Meta
}

func (agent *dynamicPathTestAgent) GetMeta() *aitypes.Meta { return &agent.meta }

func TestFilesystemToolsUseCurrentRootAfterDirectoryChanges(t *testing.T) {
	firstRoot, secondRoot := t.TempDir(), t.TempDir()
	previousDirectory := filepath.Join(firstRoot, "a-long-nested-directory-from-the-original-terminal-pane")
	if err := os.MkdirAll(previousDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	agent := &dynamicPathTestAgent{
		directoryTestAgent: directoryTestAgent{
			pathTestAgent: pathTestAgent{projectRoot: firstRoot}, renderer: directoryTestRenderer{},
		},
		meta: aitypes.Meta{Pwd: previousDirectory},
	}
	writer := &Write{agent: agent}
	reader := &ReadFiles{agent: agent}
	for _, root := range []string{firstRoot, secondRoot} {
		agent.projectRoot = root
		result, err := writer.writeFiles(context.Background(), "-- note.md --\n"+root+"\n")
		if err != nil || strings.Contains(result, "ERROR") {
			t.Fatalf("write in current root failed: %q, %v", result, err)
		}
		data, err := os.ReadFile(filepath.Join(root, "note.md"))
		if err != nil || strings.TrimSpace(string(data)) != root {
			t.Fatalf("file not written in current root %q: %q, %v", root, data, err)
		}
		result, err = reader.callFiles(context.Background(), []string{"note.md"})
		if err != nil || !strings.Contains(result, root) {
			t.Fatalf("read did not use current root %q: %q, %v", root, result, err)
		}
		if _, err := resolveWorkspacePath(agent, filepath.Join(firstRoot, "note.md")); root != firstRoot && err == nil {
			t.Fatal("old project root remained inside the write sandbox after changing roots")
		}
	}
}

func TestResolveWorkspacePathAllowsMissingDescendantsWithoutCreatingThem(t *testing.T) {
	root := t.TempDir()
	agent := &pathTestAgent{projectRoot: root}
	want := filepath.Join(root, "nested", "new", "note.md")
	got, err := resolveWorkspacePath(agent, filepath.Join("nested", "new", "note.md"))
	if err != nil || got != want {
		t.Fatalf("resolveWorkspacePath() = %q, %v; want %q", got, err, want)
	}
	if _, err := os.Stat(filepath.Dir(want)); !os.IsNotExist(err) {
		t.Fatalf("path resolution created a directory: %v", err)
	}
}

func TestResolveWorkspacePathRejectsMissingDescendantsThroughOutsideSymlink(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	link := filepath.Join(root, "outside")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	agent := &pathTestAgent{projectRoot: root}
	if _, err := resolveWorkspacePath(agent, filepath.Join("outside", "new", "note.md")); err == nil {
		t.Fatal("resolved a missing path through an outside symlink")
	}
}

func TestWriteFilesCreatesMissingParentDirectories(t *testing.T) {
	root := t.TempDir()
	agent := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	writer := &Write{agent: agent}
	result, err := writer.writeFiles(context.Background(), "-- nested/new/note.md --\ncreated\n")
	if err != nil || strings.Contains(result, "ERROR") {
		t.Fatalf("writeFiles() = %q, %v", result, err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "nested", "new", "note.md"))
	if err != nil || string(contents) != "created\n" {
		t.Fatalf("created file = %q, %v", contents, err)
	}
}

func TestWriteFileAuthorizationDescribesMissingParentDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "existing"), 0755); err != nil {
		t.Fatal(err)
	}
	agent := &directoryTestAgent{
		pathTestAgent: pathTestAgent{projectRoot: root},
		renderer:      directoryTestRenderer{},
	}
	writer := &Write{agent: agent}
	archive := "-- new/nested/file.md --\nnew\n-- existing/file.md --\nexisting\n"
	description := writer.CallDescription(context.Background(), &writeFileInputT{Archive: archive})
	if !strings.Contains(description, `new/nested/file\.md<br/>(creating directories too)`) {
		t.Fatalf("authorization omitted the conditional directory-creation note: %s", description)
	}
	if !strings.Contains(description, `existing/file\.md | <pre><code>existing&#10;</code></pre>`) {
		t.Fatalf("authorization omitted the existing file path or contents: %s", description)
	}
	if strings.Contains(description, `existing/file\.md<br/>(creating directories too)`) {
		t.Fatalf("authorization incorrectly marked an existing parent for creation: %s", description)
	}
}

func TestApplyPatchEdits(t *testing.T) {
	tests := []struct {
		name    string
		content string
		edits   []patchEditT
		want    string
		wantErr string
	}{
		{
			name:    "single edit",
			content: "one\ntwo\nthree\n",
			edits:   []patchEditT{{Old: "two", New: "2"}},
			want:    "one\n2\nthree\n",
		},
		{
			name:    "sequential edits",
			content: "alpha\nbeta\n",
			edits:   []patchEditT{{Old: "alpha", New: "gamma"}, {Old: "gamma\nbeta", New: "delta"}},
			want:    "delta\n",
		},
		{
			name:    "deletion",
			content: "keep\nremove\nkeep\n",
			edits:   []patchEditT{{Old: "remove\n", New: ""}},
			want:    "keep\nkeep\n",
		},
		{
			name:    "not found",
			content: "one\n",
			edits:   []patchEditT{{Old: "missing", New: "x"}},
			wantErr: "was not found",
		},
		{
			name:    "ambiguous match",
			content: "dup\ndup\n",
			edits:   []patchEditT{{Old: "dup", New: "x"}},
			wantErr: "matched 2 locations",
		},
		{
			name:    "empty old",
			content: "one\n",
			edits:   []patchEditT{{Old: "", New: "x"}},
			wantErr: "cannot be empty",
		},
		{
			name:    "failed edit discards earlier edits",
			content: "one\ntwo\n",
			edits:   []patchEditT{{Old: "one", New: "1"}, {Old: "missing", New: "x"}},
			wantErr: "was not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := applyPatchEdits(test.content, test.edits)

			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("applyPatchEdits() error = nil, want error containing %q", test.wantErr)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("applyPatchEdits() error = %q, want error containing %q", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("applyPatchEdits() error = %v", err)
			}
			if got != test.want {
				t.Errorf("applyPatchEdits() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPatchFileCallDescriptionRendersEmbeddedTemplate(t *testing.T) {
	patch := &patchInputT{
		File: "src/<example>.go",
		Edits: []patchEditT{
			{Old: "<old> **first** |", New: "new first"},
			{Old: "old second", New: "new second"},
		},
	}

	description := (&PatchFile{}).CallDescription(context.Background(), patch)
	for _, want := range []string{
		"File: `src/&lt;example&gt;\\.go`",
		"| Change | Text |",
		"| `-` | <pre><code>&lt;old&gt; &#42;&#42;first&#42;&#42; &#124;</code></pre> |",
		"| `+` | <pre><code>new first</code></pre> |",
		"| `-` | <pre><code>old second</code></pre> |",
		"| `+` | <pre><code>new second</code></pre> |",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("CallDescription() = %q, want it to contain %q", description, want)
		}
	}
}

func TestResolveWorkspacePath(t *testing.T) {
	pwd := t.TempDir()
	agent := &pathTestAgent{projectRoot: pwd}
	if err := os.Mkdir(filepath.Join(pwd, "sub"), 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	tests := map[string]string{
		"file.go":                            filepath.Join(pwd, "file.go"),
		"sub/file.go":                        filepath.Join(pwd, "sub", "file.go"),
		filepath.Join(pwd, "file.go"):        filepath.Join(pwd, "file.go"),
		filepath.Join(pwd, "sub", "file.go"): filepath.Join(pwd, "sub", "file.go"),
	}

	for input, want := range tests {
		got, err := resolveWorkspacePath(agent, input)
		if err != nil {
			t.Fatalf("resolveWorkspacePath(%q) error = %v", input, err)
		}
		if got != want {
			t.Errorf("resolveWorkspacePath(%q) = %q, want %q", input, got, want)
		}
	}

	for _, input := range []string{"../outside.go", filepath.Join(pwd, "..", "outside.go")} {
		if _, err := resolveWorkspacePath(agent, input); err == nil {
			t.Errorf("resolveWorkspacePath(%q) error = nil, want outside-root error", input)
		}
	}
}

func TestResolveWorkspacePathFailsWhenProjectRootIsUnavailable(t *testing.T) {
	if _, err := resolveWorkspacePath(&pathTestAgent{}, "README.md"); err == nil || !strings.Contains(err.Error(), "project root is unavailable") {
		t.Fatalf("resolveWorkspacePath() error = %v, want unavailable-root error", err)
	}
}

func TestResolveAnyPath(t *testing.T) {
	pwd := t.TempDir()
	outside := t.TempDir()
	agent := &pathTestAgent{projectRoot: pwd}

	tests := map[string]string{
		"file.go":                       filepath.Join(pwd, "file.go"),
		filepath.Join(outside, "x.png"): filepath.Join(outside, "x.png"),
	}

	for input, want := range tests {
		got, err := resolveAnyPath(agent, input)
		if err != nil {
			t.Fatalf("resolveAnyPath(%q) error = %v", input, err)
		}
		if got != want {
			t.Errorf("resolveAnyPath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWriteFileAtomicPreservesMode(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "file.txt")

	if err := os.WriteFile(filename, []byte("before"), 0640); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := writeFileAtomic(filename, []byte("after"), 0640); err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(b) != "after" {
		t.Errorf("file contents = %q, want %q", b, "after")
	}

	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0640 {
		t.Errorf("file mode = %v, want %v", info.Mode().Perm(), os.FileMode(0640))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("temp file was not cleaned up, dir contains %d entries", len(entries))
	}
}
