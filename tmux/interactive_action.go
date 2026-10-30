package tmux

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/lmorg/ttyphoon/config"
)

type interactiveActionCallback func(output []byte, exitCode int, err error)

type interactiveActionEntry struct {
	paneID     string
	scriptPath string
	callback   interactiveActionCallback
	pending    *pendingInteractiveActionResult
}

type pendingInteractiveActionResult struct {
	paneID   string
	output   []byte
	exitCode int
}

func newInteractiveActionID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:], nil
}

func (tmux *Tmux) registerInteractiveAction(id, scriptPath string, callback interactiveActionCallback) bool {
	tmux.interactiveActionMu.Lock()
	defer tmux.interactiveActionMu.Unlock()
	if tmux.interactiveActions == nil {
		tmux.interactiveActions = make(map[string]interactiveActionEntry)
	}
	if _, exists := tmux.interactiveActions[id]; exists {
		return false
	}
	tmux.interactiveActions[id] = interactiveActionEntry{scriptPath: scriptPath, callback: callback}
	return true
}

func (tmux *Tmux) bindInteractiveActionPane(id, paneID string) bool {
	tmux.interactiveActionMu.Lock()
	entry, ok := tmux.interactiveActions[id]
	if !ok {
		tmux.interactiveActionMu.Unlock()
		return false
	}
	if entry.pending != nil && entry.pending.paneID != paneID {
		delete(tmux.interactiveActions, id)
		tmux.interactiveActionMu.Unlock()
		tmux.finishInteractiveAction(entry, nil, 0, errors.New("interactive action completion came from an unexpected pane"))
		return false
	}
	entry.paneID = paneID
	pending := entry.pending
	entry.pending = nil
	if pending != nil {
		delete(tmux.interactiveActions, id)
	} else {
		tmux.interactiveActions[id] = entry
	}
	tmux.interactiveActionMu.Unlock()
	if pending != nil {
		tmux.finishInteractiveAction(entry, pending.output, pending.exitCode, nil)
	}
	return true
}

func (tmux *Tmux) discardInteractiveAction(id string) {
	tmux.interactiveActionMu.Lock()
	entry, ok := tmux.interactiveActions[id]
	delete(tmux.interactiveActions, id)
	tmux.interactiveActionMu.Unlock()
	if ok && entry.scriptPath != "" {
		_ = os.Remove(entry.scriptPath)
	}
}

func (tmux *Tmux) completeInteractiveAction(id, paneID string, output []byte, exitCode int) bool {
	tmux.interactiveActionMu.Lock()
	entry, ok := tmux.interactiveActions[id]
	if !ok {
		tmux.interactiveActionMu.Unlock()
		return false
	}
	if entry.paneID == "" {
		if entry.pending != nil {
			tmux.interactiveActionMu.Unlock()
			return false
		}
		entry.pending = &pendingInteractiveActionResult{
			paneID:   paneID,
			output:   append([]byte(nil), output...),
			exitCode: exitCode,
		}
		tmux.interactiveActions[id] = entry
		tmux.interactiveActionMu.Unlock()
		return true
	}
	if entry.paneID != paneID {
		tmux.interactiveActionMu.Unlock()
		return false
	}
	delete(tmux.interactiveActions, id)
	tmux.interactiveActionMu.Unlock()
	tmux.finishInteractiveAction(entry, output, exitCode, nil)
	return true
}

func (tmux *Tmux) finishInteractiveAction(entry interactiveActionEntry, output []byte, exitCode int, actionErr error) {
	if entry.scriptPath != "" {
		_ = os.Remove(entry.scriptPath)
	}
	if pane := tmux.panes.Get(entry.paneID); pane != nil {
		if exitCode == 0 && actionErr == nil {
			pane.actionResult.Store(1)
		} else {
			pane.actionResult.Store(2)
		}
		delay := config.PaneAutoCloseDelay()
		deadline := time.Now().Add(delay).UnixMilli()
		pane.autoCloseAt.Store(deadline)
		time.AfterFunc(delay, func() {
			if tmux.panes.Get(entry.paneID) != pane || !pane.autoCloseAt.CompareAndSwap(deadline, -1) {
				return
			}
			if err := tmux.CloseCompletedPane(entry.paneID); err != nil && tmux.renderer != nil {
				tmux.renderer.TriggerRedraw()
			}
		})
		if tmux.renderer != nil {
			tmux.renderer.TriggerRedraw()
		}
	}
	if entry.callback != nil {
		entry.callback(append([]byte(nil), output...), exitCode, actionErr)
	}
}

func (tmux *Tmux) cancelInteractiveActionsForPane(paneID string) {
	tmux.interactiveActionMu.Lock()
	var cancelled []interactiveActionEntry
	for id, entry := range tmux.interactiveActions {
		if entry.paneID == paneID {
			cancelled = append(cancelled, entry)
			delete(tmux.interactiveActions, id)
		}
	}
	tmux.interactiveActionMu.Unlock()

	for _, entry := range cancelled {
		if entry.scriptPath != "" {
			_ = os.Remove(entry.scriptPath)
		}
		if entry.callback != nil {
			entry.callback(nil, 0, errors.New("interactive action pane closed before completion"))
		}
	}
}

func (tmux *Tmux) cancelAllInteractiveActions(err error) {
	tmux.interactiveActionMu.Lock()
	entries := tmux.interactiveActions
	tmux.interactiveActions = make(map[string]interactiveActionEntry)
	tmux.interactiveActionMu.Unlock()

	for _, entry := range entries {
		if entry.scriptPath != "" {
			_ = os.Remove(entry.scriptPath)
		}
		if entry.callback != nil {
			entry.callback(nil, 0, err)
		}
	}
}
