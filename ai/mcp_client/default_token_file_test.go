package mcp_client

import (
	"path"
	"strings"
	"testing"
)

func TestDefaultTokenFile_NamespacesByWorkspace(t *testing.T) {
	const (
		server = "atlassian"
		url    = "https://mcp.atlassian.com/v1/mcp/authv2"
	)

	shared := DefaultTokenFile("", server, url)
	wsA := DefaultTokenFile("project-a", server, url)
	wsB := DefaultTokenFile("project-b", server, url)

	// The same server in different workspaces must resolve to distinct token files.
	if wsA == wsB {
		t.Fatalf("expected distinct token files per workspace, got identical: %q", wsA)
	}

	// Workspace-scoped tokens live under a per-workspace subdirectory.
	if got := path.Base(path.Dir(wsA)); got != "project-a" {
		t.Fatalf("workspace token dir = %q, want %q", got, "project-a")
	}
	if got := path.Base(path.Dir(wsB)); got != "project-b" {
		t.Fatalf("workspace token dir = %q, want %q", got, "project-b")
	}

	// The empty-workspace fallback must NOT be nested under a workspace dir.
	if path.Base(path.Dir(shared)) != "mcp-tokens" {
		t.Fatalf("empty workspace should use shared mcp-tokens dir, got %q", shared)
	}

	// All variants keep the same server-derived file name.
	name := path.Base(shared)
	if path.Base(wsA) != name || path.Base(wsB) != name {
		t.Fatalf("token file name should be stable across workspaces: %q / %q / %q", name, path.Base(wsA), path.Base(wsB))
	}
}

func TestDefaultTokenFile_SanitizesUnsafeWorkspace(t *testing.T) {
	got := DefaultTokenFile("../../etc/../weird name!", "demo", "https://example.com/mcp")

	// Path traversal and unsafe characters must be collapsed into a single
	// sanitized segment so a workspace name can never escape the token dir.
	if strings.Contains(got, "..") {
		t.Fatalf("token path must not contain traversal segments: %q", got)
	}
	dir := path.Base(path.Dir(got))
	if strings.ContainsAny(dir, "/ !") {
		t.Fatalf("workspace dir segment not sanitized: %q", dir)
	}
}
