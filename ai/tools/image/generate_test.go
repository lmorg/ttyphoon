package imagetools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
)

type fakeAgent struct {
	pwd string
	env map[string]string
}

func (a *fakeAgent) Renderer() types.Renderer         { return nil }
func (a *fakeAgent) ServiceName() string              { return "test" }
func (a *fakeAgent) GetMeta() *aitypes.Meta           { return &aitypes.Meta{Pwd: a.pwd} }
func (a *fakeAgent) EnvironmentValue(n string) string { return a.env[n] }
func (a *fakeAgent) ImageGenerationEnvironmentValue(n string) string {
	return a.env[n]
}

func TestImageEndpoint(t *testing.T) {
	tests := map[string]string{
		"":                              "https://api.openai.com/v1/images/generations",
		"https://openrouter.ai/api/v1":  "https://openrouter.ai/api/v1/images/generations",
		"https://openrouter.ai/api/v1/": "https://openrouter.ai/api/v1/images/generations",
		"  https://example.com/v1  ":    "https://example.com/v1/images/generations",
	}

	for input, want := range tests {
		if got := imageEndpoint(input); got != want {
			t.Errorf("imageEndpoint(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveOutputPath(t *testing.T) {
	pwd := t.TempDir()
	tool := &GenerateImage{agent: &fakeAgent{pwd: pwd}}

	t.Run("relative path", func(t *testing.T) {
		relative, absolute, err := tool.resolveOutputPath("images/out.png")
		if err != nil {
			t.Fatalf("resolveOutputPath() error = %v", err)
		}
		if relative != filepath.Join("images", "out.png") {
			t.Errorf("relative = %q", relative)
		}
		if absolute != filepath.Join(pwd, "images", "out.png") {
			t.Errorf("absolute = %q", absolute)
		}
	})

	t.Run("default filename", func(t *testing.T) {
		relative, absolute, err := tool.resolveOutputPath("")
		if err != nil {
			t.Fatalf("resolveOutputPath() error = %v", err)
		}
		if relative != absolute {
			t.Errorf("relative = %q, absolute = %q, want them equal", relative, absolute)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("UserHomeDir() error = %v", err)
		}
		wantDir := filepath.Join(home, app.DirName, ".images")
		if filepath.Dir(absolute) != wantDir {
			t.Errorf("directory = %q, want %q", filepath.Dir(absolute), wantDir)
		}

		base := filepath.Base(absolute)
		if !strings.HasPrefix(base, "generated-image-") || !strings.HasSuffix(base, ".png") {
			t.Errorf("filename = %q, want a timestamped png", base)
		}
	})

	t.Run("traversal is rejected", func(t *testing.T) {
		for _, name := range []string{"../escape.png", "images/../../escape.png", "/etc/passwd"} {
			if _, _, err := tool.resolveOutputPath(name); err == nil {
				t.Errorf("resolveOutputPath(%q) error = nil, want rejection", name)
			}
		}
	})
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdef", 3); got != "abc…" {
		t.Errorf("truncate() = %q", got)
	}
	if got := truncate("abc", 10); got != "abc" {
		t.Errorf("truncate() = %q", got)
	}
}
