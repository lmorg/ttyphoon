package rendererwebkit

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	goruntime "runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/lmorg/ttyphoon/utils/imageattachment"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type inputBoxHistoryEntry struct {
	Value     string                 `json:"value"`
	Variables map[string]any         `json:"variables,omitempty"`
	Images    []inputBoxHistoryImage `json:"images,omitempty"`
}

type inputBoxHistoryImage struct {
	MIMEType string `json:"mimeType"`
	Path     string `json:"path"`
	Digest   string `json:"digest"`
}

func inputBoxImageDigest(attachment types.ImageAttachment) string {
	data, _ := base64.StdEncoding.DecodeString(attachment.Base64)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func inputBoxHistoryIdentity(entry inputBoxHistoryEntry) []byte {
	entry.Images = append([]inputBoxHistoryImage(nil), entry.Images...)
	for index := range entry.Images {
		entry.Images[index].Path = ""
	}
	encoded, _ := json.Marshal(entry)
	return encoded
}

func (entry *inputBoxHistoryEntry) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*entry = inputBoxHistoryEntry{Value: value}
		return nil
	}
	type historyEntry inputBoxHistoryEntry
	var decoded historyEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*entry = inputBoxHistoryEntry(decoded)
	return nil
}

func loadInputBoxHistory(key string, fallback []string) []inputBoxHistoryEntry {
	var history []inputBoxHistoryEntry
	if key != "" {
		cache.Read(cache.NS_INPUTBOXW_HISTORY, key, &history)
	}
	for _, value := range fallback {
		if !slices.ContainsFunc(history, func(entry inputBoxHistoryEntry) bool { return entry.Value == value }) {
			history = append(history, inputBoxHistoryEntry{Value: value})
		}
	}
	return history
}

func prependInputBoxHistory(result *types.InputBoxCallbackResultT, history []inputBoxHistoryEntry) []inputBoxHistoryEntry {
	entry := inputBoxHistoryEntry{Value: result.Value, Variables: result.Variables}
	for _, attachment := range result.Images {
		entry.Images = append(entry.Images, inputBoxHistoryImage{
			MIMEType: attachment.MIMEType, Path: attachment.Path, Digest: inputBoxImageDigest(attachment),
		})
	}
	encoded := inputBoxHistoryIdentity(entry)
	updated := []inputBoxHistoryEntry{entry}
	for _, previous := range history {
		candidate := inputBoxHistoryIdentity(previous)
		if !bytes.Equal(encoded, candidate) {
			updated = append(updated, previous)
		}
	}
	return updated
}

type inputBoxCallbacksT struct {
	ok              types.InputBoxCallbackT
	cancel          types.InputBoxCallbackT
	history         []string
	cacheKey        string
	allowImagePaste bool
	draftKey        string
	historyEntries  []inputBoxHistoryEntry
}

func inputBoxDraftKey(options types.InputBoxWTOptions) string {
	if options.Confirmation {
		return ""
	}
	if options.DraftKey != "" {
		key := sha256.Sum256([]byte(options.DraftKey))
		return hex.EncodeToString(key[:])
	}
	options.History = nil
	encoded, _ := json.Marshal(options)
	key := sha256.Sum256(encoded)
	return hex.EncodeToString(key[:])
}

func loadInputBoxDraft(key string) *types.InputBoxCallbackResultT {
	if key == "" {
		return nil
	}
	var draft *types.InputBoxCallbackResultT
	cache.Read(cache.NS_INPUTBOXW_DRAFT, key, &draft)
	return draft
}

type inputBoxesT struct {
	mu        sync.Mutex
	callbacks map[int64]inputBoxCallbacksT
}

func (ib *inputBoxesT) store(ok, cancel types.InputBoxCallbackT, cacheKey string, history []string, allowImagePaste bool, draftKey string) int64 {
	id := time.Now().UnixMicro()
	ib.mu.Lock()
	if ib.callbacks == nil {
		ib.callbacks = make(map[int64]inputBoxCallbacksT)
	}
	ib.callbacks[id] = inputBoxCallbacksT{
		ok:              ok,
		cancel:          cancel,
		history:         history,
		cacheKey:        cacheKey,
		allowImagePaste: allowImagePaste,
		draftKey:        draftKey,
		historyEntries:  loadInputBoxHistory(cacheKey, history),
	}
	ib.mu.Unlock()
	return id
}

func (ib *inputBoxesT) pop(id int64) (inputBoxCallbacksT, bool) {
	ib.mu.Lock()
	cbs, ok := ib.callbacks[id]
	if ok {
		delete(ib.callbacks, id)
	}
	ib.mu.Unlock()
	return cbs, ok
}

func (wr *webkitRender) InputBoxHistoryImages(id int64, index int) ([]types.ImageAttachment, error) {
	wr.inputBoxes.mu.Lock()
	callbacks, ok := wr.inputBoxes.callbacks[id]
	wr.inputBoxes.mu.Unlock()
	if !ok || !callbacks.allowImagePaste || index < 0 || index >= len(callbacks.historyEntries) {
		return nil, fmt.Errorf("image history is unavailable for this dialog")
	}
	entry := callbacks.historyEntries[index]
	references := make([]types.ImageAttachment, 0, len(entry.Images))
	for _, attachment := range entry.Images {
		references = append(references, types.ImageAttachment{MIMEType: attachment.MIMEType, Path: attachment.Path})
	}
	images, err := imageattachment.Load(references)
	if err != nil {
		return nil, err
	}
	for imageIndex, attachment := range images {
		if inputBoxImageDigest(attachment) != entry.Images[imageIndex].Digest {
			return nil, fmt.Errorf("saved image no longer matches this history entry")
		}
	}
	return images, nil
}

type inputBoxPayload struct {
	ID              int64                          `json:"id"`
	Title           string                         `json:"title"`
	Message         string                         `json:"message"`
	DefaultValue    string                         `json:"defaultValue"`
	Placeholder     string                         `json:"placeholder"`
	History         []string                       `json:"history"`
	HistoryEntries  []inputBoxHistoryEntry         `json:"historyEntries"`
	Multiline       bool                           `json:"multiline"`
	AllowImagePaste bool                           `json:"allowImagePaste"`
	Confirmation    bool                           `json:"confirmation"`
	ConfirmLabel    string                         `json:"confirmLabel"`
	CancelLabel     string                         `json:"cancelLabel"`
	ConfirmIsDanger bool                           `json:"confirmIsDanger"`
	CanSaveDraft    bool                           `json:"canSaveDraft"`
	Draft           *types.InputBoxCallbackResultT `json:"draft,omitempty"`

	Variables []types.InputBoxWTVariables `json:"variables"`
}

// DisplayInputBoxW displays an input box with options, supporting multiline input.
func (wr *webkitRender) DisplayInputBoxW(parameters *types.InputBoxWT) {
	if parameters == nil {
		return
	}
	ok := parameters.OkFunc
	if ok == nil {
		ok = func(*types.InputBoxCallbackResultT) {}
	}
	cancel := parameters.CancelFunc
	if cancel == nil {
		cancel = func(*types.InputBoxCallbackResultT) {}
	}

	// get history

	cacheKey := parameters.Options.CacheKey
	if parameters.Options.Confirmation {
		cacheKey = ""
	} else {
		if cacheKey == "" {
			// get caller
			pc, _, _, ok := goruntime.Caller(2)
			if !ok {
				cacheKey = "DisplayInputBoxW()"
			} else {
				fn := goruntime.FuncForPC(pc)
				cacheKey = strings.Replace(fn.Name(), app.ProjectSourcePath, "", 1)
			}
			cacheKey += parameters.Options.Title
		}
	}
	historyEntries := loadInputBoxHistory(cacheKey, parameters.Options.History)
	history := make([]string, 0, len(historyEntries))
	seen := make(map[string]bool)
	for _, entry := range historyEntries {
		if !seen[entry.Value] {
			history = append(history, entry.Value)
			seen[entry.Value] = true
		}
	}

	allowImagePaste := parameters.Options.AllowImagePaste && !parameters.Options.Confirmation
	draftKey := inputBoxDraftKey(parameters.Options)
	draft := loadInputBoxDraft(draftKey)
	id := wr.inputBoxes.store(ok, cancel, cacheKey, history, allowImagePaste, draftKey)

	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalInputBox", inputBoxPayload{
			ID:              id,
			Title:           parameters.Options.Title,
			Message:         parameters.Options.Message,
			DefaultValue:    parameters.Options.Prefill,
			Placeholder:     parameters.Options.Placeholder,
			History:         history,
			HistoryEntries:  historyEntries,
			Multiline:       parameters.Options.Multiline,
			AllowImagePaste: allowImagePaste,
			Confirmation:    parameters.Options.Confirmation,
			ConfirmLabel:    parameters.Options.ConfirmLabel,
			CancelLabel:     parameters.Options.CancelLabel,
			ConfirmIsDanger: parameters.Options.ConfirmIsDanger,
			CanSaveDraft:    draftKey != "",
			Draft:           draft,
			Variables:       parameters.Options.Variables,
		})
	}
}

// InputBoxSubmit is called from wails.go when JS submits the input box.
func (wr *webkitRender) InputBoxSubmit(id int64, v *types.InputBoxCallbackResultT, isOk bool) {
	cbs, ok := wr.inputBoxes.pop(id)
	if !ok {
		return
	}
	saveDraft := v.SaveDraft && !isOk && cbs.draftKey != ""
	if (!isOk && !saveDraft) || !cbs.allowImagePaste {
		v.Images = nil
	} else if err := imageattachment.Validate(v.Images); err != nil {
		v.Images = nil
		v.SaveDraft = false
		if wr.wapp != nil {
			wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
		cbs.cancel(v)
		return
	}
	if saveDraft {
		draft := *v
		draft.SaveDraft = false
		cache.Write(cache.NS_INPUTBOXW_DRAFT, cbs.draftKey, &draft, cache.Days(30))
		cbs.cancel(v)
		return
	}
	v.SaveDraft = false
	if isOk {
		if cbs.cacheKey != "" {
			if len(v.Images) > 0 {
				stored, err := imageattachment.Store(v.Images)
				if err != nil {
					if wr.wapp != nil {
						wr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
					}
					cbs.cancel(v)
					return
				}
				v.Images = stored
			}
			history := prependInputBoxHistory(v, loadInputBoxHistory(cbs.cacheKey, cbs.history))
			cache.Write(cache.NS_INPUTBOXW_HISTORY, cbs.cacheKey, history, cache.Days(365))
		}
		if cbs.draftKey != "" {
			var emptyDraft *types.InputBoxCallbackResultT
			cache.Write(cache.NS_INPUTBOXW_DRAFT, cbs.draftKey, emptyDraft, cache.Days(30))
		}
		cbs.ok(v)
	} else {
		cbs.cancel(v)
	}
}

func (wr *webkitRender) DisplayInputBox(title, defaultValue string, ok, cancel types.InputBoxCallbackT) {
	params := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:   title,
			Prefill: defaultValue,
		},
		OkFunc:     ok,
		CancelFunc: cancel,
	}
	wr.DisplayInputBoxW(params)
}
