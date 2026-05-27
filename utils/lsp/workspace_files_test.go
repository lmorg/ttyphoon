package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

func TestNotifyDidCreateFiles(t *testing.T) {
	var buf bytes.Buffer
	tr := NewTransport(&buf, bytes.NewReader(nil))

	if err := NotifyDidCreateFiles(tr, []string{"/tmp/project/new.go"}); err != nil {
		t.Fatalf("NotifyDidCreateFiles: %v", err)
	}

	msg, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg.Method != "workspace/didCreateFiles" {
		t.Fatalf("method = %q", msg.Method)
	}

	var params struct {
		Files []struct {
			URI string `json:"uri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if len(params.Files) != 1 || params.Files[0].URI != "file:///tmp/project/new.go" {
		t.Fatalf("unexpected params: %s", string(msg.Params))
	}
}

func TestNotifyDidRenameFiles(t *testing.T) {
	var buf bytes.Buffer
	tr := NewTransport(&buf, bytes.NewReader(nil))

	if err := NotifyDidRenameFiles(tr, [][2]string{{"/tmp/project/old.go", "/tmp/project/new.go"}}); err != nil {
		t.Fatalf("NotifyDidRenameFiles: %v", err)
	}

	msg, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg.Method != "workspace/didRenameFiles" {
		t.Fatalf("method = %q", msg.Method)
	}

	var params struct {
		Files []struct {
			OldURI string `json:"oldUri"`
			NewURI string `json:"newUri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if len(params.Files) != 1 || params.Files[0].OldURI != "file:///tmp/project/old.go" || params.Files[0].NewURI != "file:///tmp/project/new.go" {
		t.Fatalf("unexpected params: %s", string(msg.Params))
	}
}

func TestNotifyDidDeleteFiles(t *testing.T) {
	var buf bytes.Buffer
	tr := NewTransport(&buf, bytes.NewReader(nil))

	if err := NotifyDidDeleteFiles(tr, []string{"/tmp/project/dead.go"}); err != nil {
		t.Fatalf("NotifyDidDeleteFiles: %v", err)
	}

	msg, err := ReadMessage(bufio.NewReader(&buf))
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if msg.Method != "workspace/didDeleteFiles" {
		t.Fatalf("method = %q", msg.Method)
	}

	var params struct {
		Files []struct {
			URI string `json:"uri"`
		} `json:"files"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if len(params.Files) != 1 || params.Files[0].URI != "file:///tmp/project/dead.go" {
		t.Fatalf("unexpected params: %s", string(msg.Params))
	}
}

func TestManagerServersForWorkspace(t *testing.T) {
	m := NewManager()
	m.servers[SessionKey{WorkspaceRoot: "/proj", LanguageID: "go"}] = &ServerProcess{}
	m.servers[SessionKey{WorkspaceRoot: "/proj", LanguageID: "python"}] = &ServerProcess{}
	m.servers[SessionKey{WorkspaceRoot: "/other", LanguageID: "go"}] = &ServerProcess{}

	servers := m.ServersForWorkspace("/proj")
	if len(servers) != 2 {
		t.Fatalf("len(servers) = %d, want 2", len(servers))
	}
}
