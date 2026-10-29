package prompts

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/utils/imageattachment"
)

func TestImageAttachmentsBecomeMultimodalUserParts(t *testing.T) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	images := []aitypes.ImageAttachment{{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(buffer.Bytes())}}
	message := buildUserMessageWithImages("Describe this image", images)
	if message.Role != schema.User || len(message.UserInputMultiContent) != 2 {
		t.Fatalf("not a multimodal user message: %+v", message)
	}
	parts := message.UserInputMultiContent
	if parts[0].Text != "Describe this image" || parts[1].Type != schema.ChatMessagePartTypeImageURL || parts[1].Image == nil {
		t.Fatalf("missing text or image parts: %+v", parts)
	}
	if parts[1].Image.MIMEType != "image/png" || parts[1].Image.Base64Data == nil || *parts[1].Image.Base64Data != images[0].Base64 {
		t.Fatal("model did not receive the image bytes")
	}
	if plain := buildUserMessageWithImages("Text only", nil); plain.Content != "Text only" || len(plain.UserInputMultiContent) != 0 {
		t.Fatal("text-only prompts changed")
	}
}

func TestAskMessagesApplySelectedEffortGuidance(t *testing.T) {
	tests := []struct {
		name   string
		effort string
		want   string
	}{
		{name: "quick", effort: "Quick", want: "Surface a consequential unresolved preference early"},
		{name: "balanced", effort: "Balanced", want: "Examine relevant context and evidence"},
		{name: "thorough", effort: "Thorough", want: "Explore relevant alternatives, edge cases, dependencies, and downstream effects"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			agt := &agent.Agent{Meta: &aitypes.Meta{Variables: map[string]any{
				EffortVariableName: test.effort,
			}}}
			messages := GetAskMessages(agt, "Inspect the current project")
			if len(messages) == 0 || !strings.Contains(messages[0].Content, test.want) {
				t.Fatalf("system prompt does not contain %q for effort %q", test.want, test.effort)
			}
		})
	}
}

func TestAskMessagesDefaultToQuickEffortGuidance(t *testing.T) {
	agt := &agent.Agent{Meta: &aitypes.Meta{}}
	messages := GetAskMessages(agt, "Inspect the current project")
	if len(messages) == 0 || !strings.Contains(messages[0].Content, "Surface a consequential unresolved preference early") {
		t.Fatal("system prompt does not use quick effort guidance when no effort is selected")
	}
}

func TestSavedImagePathsAreMappedToTheirMultimodalAttachments(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), `home with "quotes" and spaces`))
	var images []aitypes.ImageAttachment
	for index := range 2 {
		var buffer bytes.Buffer
		if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, index+2, 2))); err != nil {
			t.Fatal(err)
		}
		images = append(images, aitypes.ImageAttachment{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(buffer.Bytes())})
	}
	stored, err := imageattachment.Store(images)
	if err != nil {
		t.Fatal(err)
	}
	message := buildUserMessageWithImages("Edit these images", stored)
	parts := message.UserInputMultiContent
	if len(parts) != 5 || parts[0].Text != "Edit these images" {
		t.Fatalf("missing attachment path parts: %+v", parts)
	}
	for index, attachment := range stored {
		label, picture := parts[1+index*2], parts[2+index*2]
		if label.Type != schema.ChatMessagePartTypeText || !strings.Contains(label.Text, "generateImage") || !strings.Contains(label.Text, "inputImage") {
			t.Fatalf("missing direct tool instructions: %q", label.Text)
		}
		lines := strings.Split(label.Text, "\n")
		if len(lines) < 2 {
			t.Fatalf("missing structured file reference: %q", label.Text)
		}
		var reference struct {
			Attachment int    `json:"attachment"`
			InputImage string `json:"inputImage"`
		}
		if err := json.Unmarshal([]byte(lines[1]), &reference); err != nil || reference.Attachment != index+1 || reference.InputImage != attachment.Path {
			t.Fatalf("wrong attachment/path association: %+v, %v", reference, err)
		}
		if picture.Image == nil || picture.Image.Base64Data == nil || *picture.Image.Base64Data != attachment.Base64 {
			t.Fatal("path label was not followed by its original image")
		}
		if strings.Contains(label.Text, attachment.Base64) {
			t.Fatal("base64 leaked into attachment path text")
		}
	}
}
