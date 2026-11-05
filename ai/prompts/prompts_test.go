package prompts

import (
	"bytes"
	"context"
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
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/utils/imageattachment"
)

type fakeSkillSelectionRunner struct {
	response string
	err      error
	calls    int
	messages []*schema.Message
}

func (runner *fakeSkillSelectionRunner) RunSkillSelection(_ context.Context, messages []*schema.Message) (string, error) {
	runner.calls++
	runner.messages = messages
	return runner.response, runner.err
}

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
		name     string
		effort   string
		want     string
		recovery string
	}{
		{name: "quick", effort: "Quick", want: "Surface a consequential unresolved preference early", recovery: "do not retry with alternative approaches"},
		{name: "balanced", effort: "Balanced", want: "Examine relevant context and evidence", recovery: "at most one distinct, safe alternative"},
		{name: "thorough", effort: "Thorough", want: "Explore relevant alternatives, edge cases, dependencies, and downstream effects", recovery: "up to three distinct, safe alternatives"},
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
			if !strings.Contains(messages[0].Content, test.recovery) {
				t.Fatalf("system prompt does not contain recovery budget %q for effort %q", test.recovery, test.effort)
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
	if !strings.Contains(messages[0].Content, "do not retry with alternative approaches") || !strings.Contains(messages[0].Content, "use askUser promptly for the help needed instead of extending the loop") {
		t.Fatal("system prompt does not use Quick effort's stalled-task recovery budget")
	}
}

func TestSelectSkillsUsesDescriptionCatalogAndAllowlistedIDs(t *testing.T) {
	available := skills.Skills{
		{Name: "review", Description: "Inspect changes for defects", Prompt: "PRIVATE FULL SKILL INSTRUCTIONS"},
		{Name: "release", Description: "Prepare a release", Prompt: "RELEASE INSTRUCTIONS"},
	}
	runner := &fakeSkillSelectionRunner{response: "[\"review\",\"unknown\",\"review\"]"}
	selected := selectSkills(context.Background(), "Review this patch", available, runner)
	if runner.calls != 1 || len(selected) != 1 || selected[0] != available[0] {
		t.Fatalf("selected skills = %#v, router calls = %d", selected, runner.calls)
	}
	if len(runner.messages) != 2 || !strings.Contains(runner.messages[1].Content, "Inspect changes for defects") || !strings.Contains(runner.messages[1].Content, "Review this patch") {
		t.Fatalf("router did not receive request and catalog metadata: %+v", runner.messages)
	}
	if strings.Contains(runner.messages[1].Content, "PRIVATE FULL SKILL INSTRUCTIONS") {
		t.Fatal("router received full skill instructions instead of descriptions")
	}
}

func TestSelectSkillsHonoursExplicitMenuCommandWithoutRouting(t *testing.T) {
	selectedSkill := &skills.SkillT{Name: "review", Prompt: "Review instructions"}
	runner := &fakeSkillSelectionRunner{response: "[]"}
	selected := selectSkills(context.Background(), "/review inspect the diff", skills.Skills{selectedSkill}, runner)
	if runner.calls != 0 || len(selected) != 1 || selected[0] != selectedSkill {
		t.Fatalf("explicit skill selection = %#v, router calls = %d", selected, runner.calls)
	}
}

func TestSelectSkillsHonoursInvocationControls(t *testing.T) {
	userDisabled := false
	manualOnly := &skills.SkillT{Name: "manual", DisableModelInvocation: true}
	modelOnly := &skills.SkillT{Name: "automatic", UserInvocable: &userDisabled}
	available := skills.Skills{manualOnly, modelOnly}

	runner := &fakeSkillSelectionRunner{response: `["manual","automatic"]`}
	selected := selectSkills(context.Background(), "Review this", available, runner)
	if runner.calls != 1 || len(selected) != 1 || selected[0] != modelOnly {
		t.Fatalf("automatic selection = %#v, calls=%d; want automatic skill only", selected, runner.calls)
	}
	if strings.Contains(runner.messages[1].Content, `"id":"manual"`) {
		t.Fatal("manual-only skill was exposed to automatic routing")
	}

	runner = &fakeSkillSelectionRunner{response: "[]"}
	selected = selectSkills(context.Background(), "/manual do it", available, runner)
	if runner.calls != 0 || len(selected) != 1 || selected[0] != manualOnly {
		t.Fatalf("explicit manual skill = %#v, router calls=%d", selected, runner.calls)
	}

	runner = &fakeSkillSelectionRunner{response: `["model-only"]`}
	selected = selectSkills(context.Background(), "/automatic do it", available, runner)
	if runner.calls != 0 || len(selected) != 0 {
		t.Fatalf("user-hidden skill was explicitly selected: %#v, router calls=%d", selected, runner.calls)
	}
}

func TestSelectSkillsFallsBackWhenRouterReturnsNoSkillsOrInvalidOutput(t *testing.T) {
	available := skills.Skills{{Name: "review", Description: "Review changes"}}
	for _, response := range []string{"[]", "not JSON"} {
		runner := &fakeSkillSelectionRunner{response: response}
		if selected := selectSkills(context.Background(), "Explain this", available, runner); len(selected) != 0 {
			t.Errorf("response %q selected skills %#v, want normal-prompt fallback", response, selected)
		}
	}
}

func TestGetAskMessagesWithMultipleSkillsCombinesInstructions(t *testing.T) {
	agent := &agent.Agent{Meta: &aitypes.Meta{}}
	selected := skills.Skills{
		{Name: "review", Directory: "/skills/review", Prompt: "Review instructions"},
		{Name: "go", Directory: "/skills/go", Prompt: "Go-specific instructions"},
	}
	messages := GetAskMessagesWithSkills(agent, "Review this patch", selected)
	if len(messages) != 2 || messages[0].Role != schema.System || messages[1].Role != schema.User {
		t.Fatalf("skill prompt messages = %+v", messages)
	}
	for _, expected := range []string{"Review instructions", "Go-specific instructions", "# Additional skill: go", "Review this patch"} {
		if !strings.Contains(messages[1].Content, expected) {
			t.Errorf("combined skill prompt missing %q: %s", expected, messages[1].Content)
		}
	}
	if agent.Meta.SkillResources["review"] != "/skills/review" || agent.Meta.SkillResources["go"] != "/skills/go" {
		t.Fatalf("active skill resources = %#v", agent.Meta.SkillResources)
	}
}

func TestNormalAskClearsSkillResourceScope(t *testing.T) {
	agt := &agent.Agent{Meta: &aitypes.Meta{SkillResources: map[string]string{"old-skill": "/old/skill"}}}
	normalAskMessages(agt, "ordinary request")
	if len(agt.Meta.SkillResources) != 0 {
		t.Fatalf("ordinary prompt retained skill resource scope: %#v", agt.Meta.SkillResources)
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
