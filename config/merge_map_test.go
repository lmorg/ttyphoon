package config

import "testing"

func TestMergeMapMergesNestedMapsAndOverwritesValues(t *testing.T) {
	older := map[string]any{
		"name": "old",
		"nested": map[string]any{
			"keep":    true,
			"replace": "old",
		},
		"list": []any{"old"},
	}
	newer := map[string]any{
		"name": "new",
		"nested": map[string]any{
			"add":     "value",
			"replace": "new",
		},
		"list": []any{"new"},
	}

	got := MergeMap(older, newer)
	if got["name"] != "new" {
		t.Fatalf("name = %v, want new", got["name"])
	}
	if got["list"].([]any)[0] != "new" {
		t.Fatalf("list = %v, want replacement", got["list"])
	}

	nested := got["nested"].(map[string]any)
	if nested["keep"] != true || nested["add"] != "value" || nested["replace"] != "new" {
		t.Fatalf("nested = %#v, want merged values", nested)
	}
}

func TestMergeMapAcceptsNestedMapWithDifferentKeyType(t *testing.T) {
	older := map[string]any{
		"nested": map[string]any{"keep": "yes"},
	}
	newer := map[string]any{
		"nested": map[any]any{"add": "value"},
	}

	MergeMap(older, newer)
	nested := older["nested"].(map[string]any)
	if nested["keep"] != "yes" || nested["add"] != "value" {
		t.Fatalf("nested = %#v, want merged values", nested)
	}
}

func TestMergeMapHandlesNilOlderMap(t *testing.T) {
	got := MergeMap(nil, map[string]any{"key": "value"})
	if got["key"] != "value" {
		t.Fatalf("got = %#v, want key=value", got)
	}
}

func TestMergeMapMergesStructPointerValues(t *testing.T) {
	older := map[string]*AIServiceT{
		"service": {
			Label:        "Service",
			Provider:     "openai",
			DefaultModel: "gpt-4",
			Env:          map[string]string{"OPENAI_API_KEY": "key"},
		},
	}
	newer := map[string]*AIServiceT{
		"service": {
			Description:        "Updated description",
			DefaultModel:       "gpt-5",
			SummariseModelYaml: "gpt-5-mini",
			Env:                map[string]string{"OPENAI_BASE_URL": "https://example.test"},
		},
	}

	MergeMap(older, newer)
	service := older["service"]
	if service.Label != "Service" || service.Provider != "openai" || service.DefaultModel != "gpt-5" {
		t.Fatalf("service identity/model = %#v, want preserved identity and updated model", service)
	}
	if service.Description != "Updated description" || service.SummariseModelYaml != "gpt-5-mini" {
		t.Fatalf("service descriptions = %#v, want updated values", service)
	}
	if service.Env["OPENAI_API_KEY"] != "key" || service.Env["OPENAI_BASE_URL"] != "https://example.test" {
		t.Fatalf("service env = %#v, want merged values", service.Env)
	}
}
