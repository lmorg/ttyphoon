package lsp

import (
	"context"
	"encoding/json"
	"log"
)

// ----------------------------------------------------------------------------
// LSP publishDiagnostics types
// ----------------------------------------------------------------------------

// DiagnosticSeverity mirrors LSP spec values.
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// Position is a 0-based LSP line/character pair.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a start/end position pair.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Diagnostic is one LSP diagnostic item.
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Code     any                `json:"code,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// DiagnosticsPayload is the frontend-ready payload emitted as "notesLspDiagnostics".
type DiagnosticsPayload struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// publishDiagnosticsParams mirrors the textDocument/publishDiagnostics params.
type publishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// ----------------------------------------------------------------------------
// Notification dispatcher
// ----------------------------------------------------------------------------

// DiagnosticsEmitter is called when the server pushes diagnostics.
// The implementation in frontend.go uses runtime.EventsEmit.
type DiagnosticsEmitter func(payload DiagnosticsPayload)

// DiagnosticsContentResolver returns content for a given URI when available.
type DiagnosticsContentResolver func(uri string) (content string, ok bool)

// ListenForDiagnostics reads notifications from sp until ctx is done,
// calling emitter for each textDocument/publishDiagnostics notification.
func ListenForDiagnostics(ctx context.Context, sp *ServerProcess, contentResolver DiagnosticsContentResolver, emitter DiagnosticsEmitter) {
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
			if msg.Method != "textDocument/publishDiagnostics" {
				continue
			}
			dispatchDiagnostics(msg, serverPosEnc, contentResolver, emitter)
		}
	}
}

func dispatchDiagnostics(msg *Message, serverPosEnc PositionEncoding, contentResolver DiagnosticsContentResolver, emitter DiagnosticsEmitter) {
	var params publishDiagnosticsParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		log.Printf("lsp: parse publishDiagnostics: %v", err)
		return
	}

	if serverPosEnc != PositionEncodingUTF16 && contentResolver != nil {
		if content, ok := contentResolver(params.URI); ok {
			for i := range params.Diagnostics {
				params.Diagnostics[i].Range = convertRangeAtURI(content, params.Diagnostics[i].Range, serverPosEnc, PositionEncodingUTF16)
			}
		}
	}

	emitter(DiagnosticsPayload{
		URI:         params.URI,
		Diagnostics: params.Diagnostics,
	})
}
