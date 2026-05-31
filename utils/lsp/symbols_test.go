package lsp

import (
	"encoding/json"
	"testing"
)

func TestParseDocumentSymbolsResult_DocumentSymbolHierarchy(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"name":"Person",
			"detail":"struct",
			"kind":23,
			"range":{"start":{"line":0,"character":0},"end":{"line":8,"character":1}},
			"selectionRange":{"start":{"line":0,"character":5},"end":{"line":0,"character":11}},
			"children":[
				{
					"name":"Name",
					"kind":8,
					"range":{"start":{"line":1,"character":1},"end":{"line":1,"character":10}},
					"selectionRange":{"start":{"line":1,"character":1},"end":{"line":1,"character":5}}
				}
			]
		}
	]`)

	items, err := parseDocumentSymbolsResult(raw, "file:///tmp/main.go", "type Person struct {\nName string\n}", PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(items))
	}
	if items[0].Name != "Person" || items[0].Line != 0 || items[0].Character != 5 {
		t.Fatalf("unexpected first symbol: %+v", items[0])
	}
	if items[1].Name != "Name" || items[1].ContainerName != "Person" {
		t.Fatalf("unexpected child symbol: %+v", items[1])
	}
}

func TestParseWorkspaceSymbolsResult_SymbolInformation(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"name":"beta",
			"detail":"func",
			"kind":12,
			"containerName":"alpha",
			"location":{
				"uri":"file:///tmp/project/beta.go",
				"range":{"start":{"line":2,"character":0},"end":{"line":2,"character":4}}
			}
		}
	]`)

	items, err := parseWorkspaceSymbolsResult(raw, func(uri string) (string, bool) {
		if uri == "file:///tmp/project/beta.go" {
			return "package main\n\nfunc beta() {}\n", true
		}
		return "", false
	}, PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "beta" || items[0].FilePath != "/tmp/project/beta.go" {
		t.Fatalf("unexpected workspace symbol: %#v", items[0])
	}
	if items[0].Detail != "func" || items[0].ContainerName != "alpha" {
		t.Fatalf("unexpected metadata: %#v", items[0])
	}
}

func TestParseWorkspaceSymbolsResult_WorkspaceSymbolWire(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"name":"gamma",
			"detail":"func",
			"kind":12,
			"containerName":"pkg",
			"location":{
				"uri":"file:///tmp/project/pkg/gamma.go",
				"range":{"start":{"line":4,"character":0},"end":{"line":4,"character":5}}
			}
		}
	]`)

	items, err := parseWorkspaceSymbolsResult(raw, func(uri string) (string, bool) {
		if uri == "file:///tmp/project/pkg/gamma.go" {
			return "package pkg\n\nfunc gamma() {}\n", true
		}
		return "", false
	}, PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].FilePath != "/tmp/project/pkg/gamma.go" || items[0].Line != 4 {
		t.Fatalf("unexpected parsed workspace symbol: %#v", items[0])
	}
}

func TestParseDocumentSymbolsResult_SymbolInformation(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"name":"main",
			"kind":12,
			"containerName":"",
			"location":{
				"uri":"file:///tmp/main.go",
				"range":{"start":{"line":3,"character":0},"end":{"line":6,"character":1}}
			}
		},
		{
			"name":"foreign",
			"kind":12,
			"containerName":"",
			"location":{
				"uri":"file:///tmp/other.go",
				"range":{"start":{"line":1,"character":0},"end":{"line":1,"character":3}}
			}
		}
	]`)

	items, err := parseDocumentSymbolsResult(raw, "file:///tmp/main.go", "package main", PositionEncodingUTF16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(items))
	}
	if items[0].Name != "main" || items[0].Line != 3 {
		t.Fatalf("unexpected symbol info item: %+v", items[0])
	}
}

func TestParseDocumentSymbolsResult_ConvertsServerUTF8Character(t *testing.T) {
	raw := json.RawMessage(`[
		{
			"name":"beta",
			"kind":12,
			"range":{"start":{"line":0,"character":5},"end":{"line":0,"character":9}},
			"selectionRange":{"start":{"line":0,"character":5},"end":{"line":0,"character":9}}
		}
	]`)

	items, err := parseDocumentSymbolsResult(raw, "file:///tmp/main.go", "a😀beta", PositionEncodingUTF8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(items))
	}
	if items[0].Character != 3 {
		t.Fatalf("expected converted utf-16 character=3, got %d", items[0].Character)
	}
}
