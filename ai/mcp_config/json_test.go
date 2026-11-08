package mcp_config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestServerDefaultPermissions(t *testing.T) {
	var config ConfigT
	err := json.Unmarshal([]byte(`{
		"mcp": {
			"servers": {
				"example": {
					"DefaultPermissions": {
						"invocation": "alwaysAllow",
						"subagents": "allow"
					}
				}
			}
		}
	}`), &config)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	permissions := config.Mcp.Servers["example"].DefaultPermissions
	if permissions.Invocation != "alwaysAllow" {
		t.Fatalf("invocation = %q, want alwaysAllow", permissions.Invocation)
	}
	if permissions.Subagents != "allow" {
		t.Fatalf("subagents = %q, want allow", permissions.Subagents)
	}
}

func TestOptionalServerDescriptionInBothConfigFormats(t *testing.T) {
	for _, field := range []string{"mcp", "mcp.servers"} {
		for _, description := range []string{"", `"description":"Inspect tickets",`} {
			t.Run(field+"/"+description, func(t *testing.T) {
				servers := `{"example":{` + description + `"command":"example"}}`
				var content string
				if field == "mcp" {
					content = `{"mcp":{"servers":` + servers + `}}`
				} else {
					content = `{"mcp.servers":` + servers + `}`
				}
				path := filepath.Join(t.TempDir(), "config.json")
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
				config, err := ReadJson(path)
				if err != nil {
					t.Fatal(err)
				}
				want := ""
				if description != "" {
					want = "Inspect tickets"
				}
				if config.Mcp.Servers["example"].Description != want {
					t.Fatalf("description = %q; want %q", config.Mcp.Servers["example"].Description, want)
				}
			})
		}
	}
}
