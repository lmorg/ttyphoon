package imagetools

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

const (
	defaultImageModel   = "gpt-image-1"
	defaultImageBaseURL = "https://api.openai.com/v1"
	defaultImageSize    = "1024x1024"
	imageRequestTimeout = 5 * time.Minute
)

type GenerateImage struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&GenerateImage{})
}

//go:embed generate_description.md
var generateImageDescription string

func (t *GenerateImage) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &GenerateImage{agent: agent, enabled: true}, nil
}

func (t *GenerateImage) Enabled() bool { return t.enabled }
func (t *GenerateImage) Toggle()       { t.enabled = !t.enabled }

func (t *GenerateImage) Name() string        { return "generateImage" }
func (t *GenerateImage) Path() string        { return "internal" }
func (t *GenerateImage) Description() string { return generateImageDescription }
func (t *GenerateImage) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

type generateImageInputT struct {
	Prompt  string `json:"prompt"`
	File    string `json:"file"`
	Size    string `json:"size"`
	Quality string `json:"quality"`
}

type imageRequestT struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
	N       int    `json:"n"`
}

type imageResponseT struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
		URL     string `json:"url"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (t *GenerateImage) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	var request generateImageInputT
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		return fmt.Sprintf("ERROR: input must be valid JSON matching the tool schema: %s", err), nil
	}

	if strings.TrimSpace(request.Prompt) == "" {
		return "ERROR: 'prompt' is required", nil
	}

	apiKey := t.agent.ImageGenerationEnvironmentValue("OPENAI_API_KEY")
	if strings.TrimSpace(apiKey) == "" {
		return t.fail("ERROR: no OPENAI_API_KEY configured for this service"), nil
	}

	relative, filename, err := t.resolveOutputPath(request.File)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR: %s", err)), nil
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" generating image: "+relative)

	b, err := t.requestImage(ctx, request)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR: %s", err)), nil
	}

	if err = os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", relative, err)), nil
	}

	// O_EXCL so a generated image can never clobber an existing file.
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0664)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", relative, err)), nil
	}
	defer f.Close()

	if _, err = f.Write(b); err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", relative, err)), nil
	}

	result := fmt.Sprintf("INFO: image written to '%s' (%d bytes). Reference it in your reply as ![](%s)\n", relative, len(b), relative)
	debug.Log(result)
	return result, nil
}

func (t *GenerateImage) fail(message string) string {
	t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, message)
	debug.Log(message)
	return message + "\nNo image was written to disk.\n"
}

func (t *GenerateImage) requestImage(ctx context.Context, request generateImageInputT) ([]byte, error) {
	model := t.agent.ImageGenerationEnvironmentValue("OPENAI_IMAGE_MODEL")
	if strings.TrimSpace(model) == "" {
		model = defaultImageModel
	}

	size := strings.TrimSpace(request.Size)
	if size == "" {
		size = defaultImageSize
	}

	body, err := json.Marshal(imageRequestT{
		Model:   model,
		Prompt:  request.Prompt,
		Size:    size,
		Quality: strings.TrimSpace(request.Quality),
		N:       1,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, imageRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imageEndpoint(t.agent.ImageGenerationEnvironmentValue("OPENAI_BASE_URL")), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.agent.ImageGenerationEnvironmentValue("OPENAI_API_KEY"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}

	var decoded imageResponseT
	if err = json.Unmarshal(b, &decoded); err != nil {
		return nil, fmt.Errorf("image API returned %s: %s", resp.Status, truncate(string(b), 256))
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("image API error: %s", decoded.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image API returned %s: %s", resp.Status, truncate(string(b), 256))
	}
	if len(decoded.Data) == 0 {
		return nil, fmt.Errorf("image API returned no images")
	}

	if decoded.Data[0].B64JSON != "" {
		return base64.StdEncoding.DecodeString(decoded.Data[0].B64JSON)
	}
	if decoded.Data[0].URL != "" {
		return downloadImage(ctx, decoded.Data[0].URL)
	}

	return nil, fmt.Errorf("image API response contained neither image data nor a URL")
}

func downloadImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading generated image returned %s", resp.Status)
	}

	return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
}

// resolveOutputPath returns the path for the model to quote back, and the
// absolute path to write to.
func (t *GenerateImage) resolveOutputPath(name string) (relative string, absolute string, err error) {
	pwd := t.agent.ProjectRoot()

	name = strings.TrimSpace(name)
	if name == "" {
		absolute, err = defaultImagePath()
		return absolute, absolute, err
	}

	absolute = name
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(pwd, absolute)
	}
	absolute = filepath.Clean(absolute)

	// Constrain writes to the workspace so a crafted path can't escape it.
	if pwd != "" && absolute != pwd && !strings.HasPrefix(absolute, strings.TrimSuffix(pwd, string(filepath.Separator))+string(filepath.Separator)) {
		return "", "", fmt.Errorf("'%s' is outside the working directory", name)
	}

	relative = absolute
	if pwd != "" {
		if rel, relErr := filepath.Rel(pwd, absolute); relErr == nil {
			relative = rel
		}
	}

	return relative, absolute, nil
}

// defaultImagePath is used when the caller doesn't name a file, keeping
// generated images out of the user's working directory.
func defaultImagePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	name := fmt.Sprintf("generated-image-%s.png", time.Now().Format("20060102-150405"))
	return filepath.Join(home, app.DirName, ".images", name), nil
}

func imageEndpoint(baseURL string) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultImageBaseURL
	}
	return strings.TrimSuffix(baseURL, "/") + "/images/generations"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
