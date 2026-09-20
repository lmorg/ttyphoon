package mcp_config

import (
	"encoding/json"
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
