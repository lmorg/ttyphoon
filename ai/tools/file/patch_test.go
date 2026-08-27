package filetools

import (
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

func (a *pathTestAgent) Renderer() types.Renderer                      { return nil }
func (a *pathTestAgent) ServiceName() string                           { return "test" }
func (a *pathTestAgent) GetMeta() *aitypes.Meta                        { return nil }
func (a *pathTestAgent) ProjectRoot() string                           { return a.projectRoot }
func (a *pathTestAgent) EnvironmentValue(string) string                { return "" }
func (a *pathTestAgent) ImageGenerationEnvironmentValue(string) string { return "" }

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
