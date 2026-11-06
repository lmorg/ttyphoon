package mainwindow

import (
	"strings"

	"github.com/lmorg/ttyphoon/types"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) SendIpc(eventName string, parameters map[string]string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	switch eventName {
	case "terminal-extra-tab-state":
		tabID := strings.TrimSpace(parameters["id"])
		if tabID == "" {
			return
		}

		enabled := strings.EqualFold(parameters["enabled"], "true")
		if !enabled {
			renderer.SetTerminalPaneTabs(nil)
			return
		}

		tabName := strings.TrimSpace(parameters["name"])
		if tabName == "" {
			tabName = tabID
		}

		active := strings.EqualFold(parameters["active"], "true")
		renderer.SetTerminalPaneTabs([]types.TerminalPaneTab{{
			ID:     tabID,
			Name:   tabName,
			Active: active,
		}})

	case "terminal-notify":
		message := strings.TrimSpace(parameters["message"])
		if message == "" {
			return
		}

		switch strings.ToLower(strings.TrimSpace(parameters["level"])) {
		case "error":
			renderer.DisplayNotification(types.NOTIFY_ERROR, message)
		case "warn", "warning":
			renderer.DisplayNotification(types.NOTIFY_WARN, message)
		case "debug":
			renderer.DisplayNotification(types.NOTIFY_DEBUG, message)
		default:
			renderer.DisplayNotification(types.NOTIFY_INFO, message)
		}

	case "terminal-sticky-create":
		id := strings.TrimSpace(parameters["id"])
		message := strings.TrimSpace(parameters["message"])
		if id == "" || message == "" {
			return
		}
		var notifType types.NotificationType
		switch strings.ToLower(strings.TrimSpace(parameters["level"])) {
		case "error":
			notifType = types.NOTIFY_ERROR
		case "warn", "warning":
			notifType = types.NOTIFY_WARN
		default:
			notifType = types.NOTIFY_INFO
		}
		a.notesMu.Lock()
		if existing, ok := a.notesStickies[id]; ok {
			existing.Close()
			delete(a.notesStickies, id)
		}
		a.notesMu.Unlock()
		sticky := renderer.DisplaySticky(notifType, message, func() {})
		a.notesMu.Lock()
		a.notesStickies[id] = sticky
		a.notesMu.Unlock()

	case "terminal-sticky-update":
		id := strings.TrimSpace(parameters["id"])
		message := strings.TrimSpace(parameters["message"])
		if id == "" || message == "" {
			return
		}
		a.notesMu.Lock()
		sticky, ok := a.notesStickies[id]
		a.notesMu.Unlock()
		if ok {
			sticky.SetMessage(message)
		}

	case "terminal-sticky-close":
		id := strings.TrimSpace(parameters["id"])
		if id == "" {
			return
		}
		a.notesMu.Lock()
		if sticky, ok := a.notesStickies[id]; ok {
			sticky.Close()
			delete(a.notesStickies, id)
		}
		a.notesMu.Unlock()
	}
}
