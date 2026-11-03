package rendererwebkit

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"testing"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/lmorg/ttyphoon/utils/imageattachment"
)

func TestInputboxHistoryMigratesTextAndRetainsDistinctVariableSets(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	encodedImage := base64.StdEncoding.EncodeToString(imageData.Bytes())
	images, err := imageattachment.Store([]types.ImageAttachment{{MIMEType: "image/png", Base64: encodedImage}})
	if err != nil {
		t.Fatal(err)
	}
	key := t.TempDir()
	loadInputBoxHistory(key, nil)
	cache.Write(cache.NS_INPUTBOXW_HISTORY, key, []string{"legacy prompt"}, cache.Days(365))
	history := loadInputBoxHistory(key, []string{"provided prompt"})
	if len(history) != 2 || history[0].Value != "legacy prompt" || history[0].Variables != nil || history[1].Value != "provided prompt" {
		t.Fatalf("legacy history was not retained: %+v", history)
	}
	for _, model := range []string{"first", "second", "first"} {
		result := &types.InputBoxCallbackResultT{Value: "same prompt", Variables: map[string]any{
			"model": model, "flag": false, "count": 0,
		}, Images: images, SaveDraft: true}
		history = prependInputBoxHistory(result, history)
		cache.Write(cache.NS_INPUTBOXW_HISTORY, key, history, cache.Days(365))
		history = loadInputBoxHistory(key, nil)
	}
	if len(history) != 4 || history[0].Variables["model"] != "first" || history[1].Variables["model"] != "second" || history[0].Variables["flag"] != false || history[0].Variables["count"] != float64(0) {
		t.Fatalf("distinct variable sets were not retained: %+v", history)
	}
	encoded, err := json.Marshal(history)
	if err != nil || bytes.Contains(encoded, []byte(encodedImage)) || bytes.Contains(encoded, []byte("saveDraft")) {
		t.Fatalf("non-history payload leaked: %s, %v", encoded, err)
	}
}

func TestInputboxHistoryRestoresFileBackedImagesOnlyForOpenEnabledDialogs(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	attachment := types.ImageAttachment{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(buffer.Bytes())}
	key := t.TempDir()
	loadInputBoxHistory(key, nil)
	wr := &webkitRender{}
	noop := func(*types.InputBoxCallbackResultT) {}
	for range 2 {
		id := wr.inputBoxes.store(noop, noop, key, nil, true, "")
		wr.InputBoxSubmit(id, &types.InputBoxCallbackResultT{Value: "With image", Images: []types.ImageAttachment{attachment}}, true)
	}
	history := loadInputBoxHistory(key, nil)
	if len(history) != 1 || len(history[0].Images) != 1 || history[0].Images[0].Path == "" {
		t.Fatalf("image history missing or not deduplicated: %+v", history)
	}
	encoded, _ := json.Marshal(history)
	if bytes.Contains(encoded, []byte(attachment.Base64)) {
		t.Fatal("history persisted inline image bytes")
	}
	id := wr.inputBoxes.store(noop, noop, key, nil, true, "")
	images, err := wr.InputBoxHistoryImages(id, 0)
	if err != nil || len(images) != 1 || images[0].Base64 != attachment.Base64 {
		t.Fatalf("history image not restored: %+v, %v", images, err)
	}
	if _, err := wr.InputBoxHistoryImages(id, 1); err == nil {
		t.Fatal("accepted an invalid history index")
	}
	wr.InputBoxSubmit(id, &types.InputBoxCallbackResultT{}, false)
	if _, err := wr.InputBoxHistoryImages(id, 0); err == nil {
		t.Fatal("restored history after the dialog closed")
	}
	disabledID := wr.inputBoxes.store(noop, noop, key, nil, false, "")
	if _, err := wr.InputBoxHistoryImages(disabledID, 0); err == nil {
		t.Fatal("restored images into an image-disabled dialog")
	}
}

func TestInputboxDraftIdentityAndTypedValues(t *testing.T) {
	options := types.InputBoxWTOptions{
		CacheKey: t.TempDir(), Title: "Ask AI", Prefill: "Initial prompt", Multiline: true,
		Variables: []types.InputBoxWTVariables{{Name: "count", Type: "integer", Default: "1"}},
	}
	key := inputBoxDraftKey(options)
	if draft := loadInputBoxDraft(key); draft != nil {
		t.Fatal("fresh dialog already has a draft")
	}
	options.History = []string{"Previous submission"}
	if inputBoxDraftKey(options) != key {
		t.Fatal("history changes invalidated draft identity")
	}
	options.Prefill = "Different initial prompt"
	if inputBoxDraftKey(options) == key {
		t.Fatal("different dialog parameters share a draft")
	}
	draft := &types.InputBoxCallbackResultT{Value: "Draft prompt", Variables: map[string]any{
		"count": 3, "enabled": false, "text": "", "mode": "second",
	}}
	cache.Write(cache.NS_INPUTBOXW_DRAFT, key, draft, cache.Days(30))
	restored := loadInputBoxDraft(key)
	if restored == nil || restored.Value != draft.Value || restored.Variables["count"] != float64(3) || restored.Variables["enabled"] != false || restored.Variables["text"] != "" || restored.Variables["mode"] != "second" {
		t.Fatalf("draft did not preserve typed values: %+v", restored)
	}
	options.Confirmation = true
	if inputBoxDraftKey(options) != "" {
		t.Fatal("confirmation dialogs have draft storage")
	}
}

func TestInputboxDraftKeyUsesExplicitScopeInsteadOfDialogOptions(t *testing.T) {
	options := types.InputBoxWTOptions{
		DraftKey: "ai-workspace:editor", Title: "Ask AI", Prefill: "URI prompt",
		Variables: []types.InputBoxWTVariables{{Name: "model", Type: "list", Default: "Model A"}},
	}
	key := inputBoxDraftKey(options)
	options.Title = "Ask AI about document.md"
	options.Prefill = ""
	options.Variables[0].Default = "Model B"
	if inputBoxDraftKey(options) != key {
		t.Fatal("dialog changes invalidated an explicitly scoped draft")
	}
	options.DraftKey = "ai-workspace:terminal"
	if inputBoxDraftKey(options) == key {
		t.Fatal("different workspace draft scopes share a key")
	}
}

func TestInputboxImagePasteRequiresOptInAndValidData(t *testing.T) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	valid := types.ImageAttachment{MIMEType: "image/png", Base64: base64.StdEncoding.EncodeToString(buffer.Bytes())}
	t.Run("draft retains images", func(t *testing.T) {
		key := inputBoxDraftKey(types.InputBoxWTOptions{CacheKey: t.TempDir(), AllowImagePaste: true})
		loadInputBoxDraft(key)
		wr := &webkitRender{}
		id := wr.inputBoxes.store(func(*types.InputBoxCallbackResultT) { t.Fatal("draft submitted") },
			func(*types.InputBoxCallbackResultT) {}, "", nil, true, key)
		wr.InputBoxSubmit(id, &types.InputBoxCallbackResultT{Value: "With image", Images: []types.ImageAttachment{valid}, SaveDraft: true}, false)
		if draft := loadInputBoxDraft(key); draft == nil || len(draft.Images) != 1 || draft.Images[0].Base64 != valid.Base64 {
			t.Fatalf("draft image was lost: %+v", draft)
		}
	})
	for _, test := range []struct {
		name                             string
		allow, submit, wantOK, wantImage bool
		image                            types.ImageAttachment
	}{
		{"disabled", false, true, true, false, valid},
		{"enabled", true, true, true, true, valid},
		{"cancelled", true, false, false, false, valid},
		{"invalid", true, true, false, false, types.ImageAttachment{MIMEType: "image/png", Base64: "bad"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			wr := &webkitRender{}
			var submitted *types.InputBoxCallbackResultT
			var gotOK bool
			id := wr.inputBoxes.store(func(result *types.InputBoxCallbackResultT) {
				submitted, gotOK = result, true
			}, func(result *types.InputBoxCallbackResultT) {
				submitted = result
			}, "", nil, test.allow, "")
			wr.InputBoxSubmit(id, &types.InputBoxCallbackResultT{Value: "prompt", Images: []types.ImageAttachment{test.image}}, test.submit)
			if submitted == nil || gotOK != test.wantOK || (len(submitted.Images) > 0) != test.wantImage {
				t.Fatalf("callback = %+v, ok = %v", submitted, gotOK)
			}
		})
	}
}

func TestInputboxSaveDraftDoesNotSubmitOrChangeHistory(t *testing.T) {
	options := types.InputBoxWTOptions{Title: "Draft lifecycle", CacheKey: t.TempDir()}
	draftKey := inputBoxDraftKey(options)
	loadInputBoxDraft(draftKey)
	var history []string
	cache.Read(cache.NS_INPUTBOXW_HISTORY, options.CacheKey, &history)
	cache.Write(cache.NS_INPUTBOXW_HISTORY, options.CacheKey, []string{"submitted"}, cache.Days(365))
	wr := &webkitRender{}
	var okCalls, cancelCalls int
	open := func() int64 {
		return wr.inputBoxes.store(func(*types.InputBoxCallbackResultT) { okCalls++ },
			func(*types.InputBoxCallbackResultT) { cancelCalls++ }, options.CacheKey, []string{"submitted"}, false, draftKey)
	}
	wr.InputBoxSubmit(open(), &types.InputBoxCallbackResultT{
		Value: "unfinished", Variables: map[string]any{"flag": false, "count": 0}, SaveDraft: true,
	}, false)
	if okCalls != 0 || cancelCalls != 1 {
		t.Fatalf("Save Draft submitted an action: OK=%d cancel=%d", okCalls, cancelCalls)
	}
	if draft := loadInputBoxDraft(draftKey); draft == nil || draft.Value != "unfinished" || draft.SaveDraft {
		t.Fatalf("draft was not saved cleanly: %+v", draft)
	}
	cache.Read(cache.NS_INPUTBOXW_HISTORY, options.CacheKey, &history)
	if len(history) != 1 || history[0] != "submitted" {
		t.Fatalf("draft entered submission history: %v", history)
	}
	wr.InputBoxSubmit(open(), &types.InputBoxCallbackResultT{Value: "cancelled edits"}, false)
	if draft := loadInputBoxDraft(draftKey); draft == nil || draft.Value != "unfinished" {
		t.Fatal("Cancel changed the existing draft")
	}
	wr.InputBoxSubmit(open(), &types.InputBoxCallbackResultT{Value: "finished", Variables: map[string]any{"flag": false, "count": 0}}, true)
	if draft := loadInputBoxDraft(draftKey); draft != nil || okCalls != 1 {
		t.Fatal("accepted submission did not clear the draft")
	}
	stored := loadInputBoxHistory(options.CacheKey, nil)
	if len(stored) != 2 || stored[0].Value != "finished" || stored[0].Variables["flag"] != false || stored[0].Variables["count"] != float64(0) || stored[1].Value != "submitted" {
		t.Fatalf("accepted submission did not migrate text history: %+v", stored)
	}
}
