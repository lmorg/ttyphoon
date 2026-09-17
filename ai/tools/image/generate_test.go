package imagetools

import (
	"context"
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

func (a *fakeAgent) Renderer() types.Renderer { return nil }
func (a *fakeAgent) ServiceName() string      { return "test" }
func (a *fakeAgent) GetMeta() *aitypes.Meta   { return &aitypes.Meta{Pwd: a.pwd} }
func (a *fakeAgent) ProjectRoot() string      { return a.pwd }
func (a *fakeAgent) RequestWritePermission(context.Context, string) error {
	return nil
}
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
		if got := imageEndpoint(input, "generations"); got != want {
			t.Errorf("imageEndpoint(%q) = %q, want %q", input, got, want)
		}
	}

	if got := imageEndpoint("", "edits"); got != "https://api.openai.com/v1/images/edits" {
		t.Errorf("imageEndpoint(edits) = %q", got)
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

func TestIsOpenRouterBaseURL(t *testing.T) {
	tests := map[string]bool{
		"":                             false,
		"https://api.openai.com/v1":    false,
		"https://openrouter.ai/api/v1": true,
		"https://OpenRouter.ai/api/v1": true,
	}
	for input, want := range tests {
		if got := isOpenRouterBaseURL(input); got != want {
			t.Errorf("isOpenRouterBaseURL(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestEncodeImageDataURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if err := os.WriteFile(path, pngHeader, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := encodeImageDataURL(path)
	if err != nil {
		t.Fatalf("encodeImageDataURL() error = %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("encodeImageDataURL() = %q, want image/png data URL", got)
	}
}

func TestGenerateImageObservationRecordsOutputAndInputImage(t *testing.T) {
	observation := (&GenerateImage{}).Observation(
		`{"prompt":"make it brighter","inputImage":"before.png"}`,
		"INFO: image written to 'after.png' (123 bytes). Reference it in your reply as ![](after.png)\n",
		nil,
	)

	if observation.Tool != "generateImage" || observation.Status != "ok" {
		t.Fatalf("observation = %+v, want generateImage ok", observation)
	}
	if strings.Join(observation.Inputs, ",") != "before.png" {
		t.Fatalf("Inputs = %#v, want before.png", observation.Inputs)
	}
	if strings.Join(observation.Outputs, ",") != "after.png" {
		t.Fatalf("Outputs = %#v, want after.png", observation.Outputs)
	}
	if observation.Counts["images"] != 1 {
		t.Fatalf("images count = %d, want 1", observation.Counts["images"])
	}
}

func TestResolveInputImagePath(t *testing.T) {
	pwd := t.TempDir()
	tool := &GenerateImage{agent: &fakeAgent{pwd: pwd}}

	existing := filepath.Join(pwd, "photo.png")
	if err := os.WriteFile(existing, []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Run("existing file within workspace", func(t *testing.T) {
		got, err := tool.resolveInputImagePath("photo.png")
		if err != nil {
			t.Fatalf("resolveInputImagePath() error = %v", err)
		}
		if got != existing {
			t.Errorf("resolveInputImagePath() = %q, want %q", got, existing)
		}
	})

	t.Run("missing file is rejected", func(t *testing.T) {
		if _, err := tool.resolveInputImagePath("missing.png"); err == nil {
			t.Error("resolveInputImagePath() error = nil, want rejection")
		}
	})

	t.Run("traversal outside workspace is rejected", func(t *testing.T) {
		if _, err := tool.resolveInputImagePath("../escape.png"); err == nil {
			t.Error("resolveInputImagePath() error = nil, want rejection")
		}
	})

	t.Run("images dir is allowed", func(t *testing.T) {
		dir, err := defaultImagesDir()
		if err != nil {
			t.Fatalf("defaultImagesDir() error = %v", err)
		}
		if err = os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		generated := filepath.Join(dir, "generated-image-test.png")
		if err = os.WriteFile(generated, []byte("data"), 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		defer os.Remove(generated)

		got, err := tool.resolveInputImagePath(generated)
		if err != nil {
			t.Fatalf("resolveInputImagePath() error = %v", err)
		}
		if got != generated {
			t.Errorf("resolveInputImagePath() = %q, want %q", got, generated)
		}
	})
}
