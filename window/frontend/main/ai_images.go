package mainwindow

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/utils/file"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

const (
	maxAIImageAttachments = 6
	maxAIImageBytes       = 8 * 1024 * 1024
)

var (
	rxMarkdownImage = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	rxDataURLImage  = regexp.MustCompile(`^data:(image/[a-zA-Z0-9.+-]+);base64,(.+)$`)
)

// collectAIImageAttachments finds images to attach to an AI prompt: either the
// whole contents is a single data URL (Notes image viewer), or contents is
// markdown with embedded image references to resolve and read from disk.
func (a *WApp) collectAIImageAttachments(contents string) []aitypes.ImageAttachment {
	if m := rxDataURLImage.FindStringSubmatch(strings.TrimSpace(contents)); m != nil {
		return []aitypes.ImageAttachment{{MIMEType: m[1], Base64: m[2]}}
	}

	matches := rxMarkdownImage.FindAllStringSubmatch(contents, -1)
	if len(matches) == 0 {
		return nil
	}

	images := make([]aitypes.ImageAttachment, 0, len(matches))
	for _, match := range matches {
		if len(images) >= maxAIImageAttachments {
			break
		}

		ref := strings.TrimSpace(match[1])
		if dm := rxDataURLImage.FindStringSubmatch(ref); dm != nil {
			images = append(images, aitypes.ImageAttachment{MIMEType: dm[1], Base64: dm[2]})
			continue
		}
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			continue // remote images are not fetched for AI attachments
		}

		if attachment, ok := a.readImageAttachment(ref); ok {
			images = append(images, attachment)
		}
	}

	return images
}

func (a *WApp) readImageAttachment(path string) (aitypes.ImageAttachment, bool) {
	ext := strings.ToLower(rxExtension.FindString(path))
	if ext == "" {
		return aitypes.ImageAttachment{}, false
	}

	resolvedPath := a.resolveMarkdownAssetPath(path)

	stat, err := os.Stat(resolvedPath)
	if err != nil || stat.Size() > maxAIImageBytes {
		return aitypes.ImageAttachment{}, false
	}

	b, err := os.ReadFile(resolvedPath)
	if err != nil {
		return aitypes.ImageAttachment{}, false
	}

	return aitypes.ImageAttachment{
		MIMEType: imageMime(ext),
		Base64:   base64.StdEncoding.EncodeToString(b),
	}, true
}

// AskAIImage starts an image-aware document explanation without placing the
// image data in the document text or session log metadata.
func (a *WApp) AskAIImage(filename, dataURL string) {
	match := rxDataURLImage.FindStringSubmatch(strings.TrimSpace(dataURL))
	if match == nil {
		log.Printf("[debug] WApp AskAIImage: invalid image data URL")
		return
	}

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}
	tile := renderer.ActiveTile()
	if tile == nil {
		return
	}
	agt := agent.Get(tile.Id())
	if agt == nil {
		return
	}

	imageName := strings.TrimSpace(filename)
	if imageName == "" {
		imageName = "Image"
	}

	imagePath, err := saveAIImageUpload(match[1], match[2])
	if err != nil {
		log.Printf("[debug] WApp AskAIImage: save upload: %v", err)
		return
	}
	imageMarkdown := fmt.Sprintf("![uploaded %s](%s)", imagePath, imagePath)
	ai.ExplainDoc(agt, imageName, "Image attachment: "+imageName+"\n\n"+imageMarkdown, []aitypes.ImageAttachment{{
		MIMEType: match[1],
		Base64:   match[2],
	}})
}

func saveAIImageUpload(mimeType, encoded string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	extension := ".bin"
	if slash := strings.LastIndex(mimeType, "/"); slash >= 0 {
		extension = "." + strings.ToLower(strings.TrimLeft(mimeType[slash+1:], "."))
		if extension == ".svg+xml" {
			extension = ".svg"
		}
	}
	if !regexp.MustCompile(`^\.[a-z0-9]+$`).MatchString(extension) {
		extension = ".bin"
	}

	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	dir := aiImagesDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("uploaded-image-%d%s", time.Now().UnixNano(), extension))
	if err := os.WriteFile(path, payload, 0o664); err != nil {
		return "", err
	}
	return path, nil
}

func aiImagesDir(home string) string {
	return filepath.Join(file.DocumentsDir(home), ".images")
}
