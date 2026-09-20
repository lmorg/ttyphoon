package lsp

import (
	"context"
	"encoding/json"
	"log"
)

// LspLogPayload is emitted to frontend as notesLspLog.
type LspLogPayload struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}

// LspProgressPayload is emitted to frontend as notesLspProgress.
type LspProgressPayload struct {
	Token       string `json:"token"`
	Kind        string `json:"kind"`
	Title       string `json:"title,omitempty"`
	Message     string `json:"message,omitempty"`
	Percentage  int    `json:"percentage,omitempty"`
	Cancellable bool   `json:"cancellable,omitempty"`
}

type logMessageParams struct {
	Type    int    `json:"type"`
	Message string `json:"message"`
}

type progressParams struct {
	Token json.RawMessage `json:"token"`
	Value struct {
		Kind        string `json:"kind"`
		Title       string `json:"title,omitempty"`
		Message     string `json:"message,omitempty"`
		Percentage  *int   `json:"percentage,omitempty"`
		Cancellable bool   `json:"cancellable,omitempty"`
	} `json:"value"`
}

// ListenForNotifications consumes server notifications and dispatches known
// payloads to corresponding emitters.
func ListenForNotifications(
	ctx context.Context,
	sp *ServerProcess,
	contentResolver DiagnosticsContentResolver,
	diagnosticsEmitter DiagnosticsEmitter,
	progressEmitter func(payload LspProgressPayload),
	logEmitter func(payload LspLogPayload),
) {
	ch := sp.Notifications()
	serverPosEnc := sp.PositionEncoding()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}

			switch msg.Method {
			case "textDocument/publishDiagnostics":
				if diagnosticsEmitter != nil {
					dispatchDiagnostics(msg, serverPosEnc, contentResolver, diagnosticsEmitter)
				}
			case "window/logMessage":
				if logEmitter != nil {
					if payload, ok := parseLogMessage(msg.Params); ok {
						logEmitter(payload)
					}
				}
			case "$/progress":
				if progressEmitter != nil {
					if payload, ok := parseProgressMessage(msg.Params); ok {
						progressEmitter(payload)
					}
				}
			}
		}
	}
}

func parseLogMessage(raw json.RawMessage) (LspLogPayload, bool) {
	var params logMessageParams
	if err := json.Unmarshal(raw, &params); err != nil {
		log.Printf("[error] lsp: parse window/logMessage: %v", err)
		return LspLogPayload{}, false
	}

	if params.Message == "" {
		return LspLogPayload{}, false
	}

	return LspLogPayload{Type: params.Type, Message: params.Message}, true
}

func parseProgressMessage(raw json.RawMessage) (LspProgressPayload, bool) {
	var params progressParams
	if err := json.Unmarshal(raw, &params); err != nil {
		log.Printf("[error] lsp: parse $/progress: %v", err)
		return LspProgressPayload{}, false
	}

	if params.Value.Kind == "" {
		return LspProgressPayload{}, false
	}

	payload := LspProgressPayload{
		Token:       progressTokenString(params.Token),
		Kind:        params.Value.Kind,
		Title:       params.Value.Title,
		Message:     params.Value.Message,
		Cancellable: params.Value.Cancellable,
	}

	if params.Value.Percentage != nil {
		payload.Percentage = *params.Value.Percentage
	}

	return payload, true
}

func progressTokenString(token json.RawMessage) string {
	if len(token) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(token, &s); err == nil {
		return s
	}

	var n float64
	if err := json.Unmarshal(token, &n); err == nil {
		return string(token)
	}

	return string(token)
}
