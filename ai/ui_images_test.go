package ai

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

func TestInputboxImagesKeepBase64OutOfVisibleMetadata(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home with spaces"))
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	attachment := aitypes.ImageAttachment{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(buffer.Bytes())}
	agt := &agent.Agent{Meta: &aitypes.Meta{OutputBlock: "Existing context"}}
	stored, err := prepareInputboxImages(agt, []aitypes.ImageAttachment{attachment})
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].Base64 != attachment.Base64 {
		t.Fatalf("model attachment was lost: %+v", stored)
	}
	if _, err := os.Stat(stored[0].Path); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(agt.Meta.OutputBlock, "Existing context") || !strings.Contains(agt.Meta.OutputBlock, "![attached image]") || !strings.Contains(agt.Meta.OutputBlock, "%20") {
		t.Fatalf("missing safe image metadata: %q", agt.Meta.OutputBlock)
	}
	if strings.Contains(agt.Meta.OutputBlock, attachment.Base64) || strings.Contains(agt.Meta.OutputBlock, "data:image") {
		t.Fatal("base64 leaked into visible metadata")
	}
}
