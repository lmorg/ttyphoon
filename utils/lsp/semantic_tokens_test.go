package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
)

func TestConvertSemanticTokensToUTF16_ConvertsCharacterAndLength(t *testing.T) {
	// "a😀beta": the emoji is 4 UTF-8 bytes but 2 UTF-16 code units, so a token
	// starting at UTF-8 offset 5 starts at UTF-16 offset 3.
	got := convertSemanticTokensToUTF16([]int{0, 5, 4, 2, 0}, "a😀beta", PositionEncodingUTF8)

	assertTokenData(t, got, []int{0, 3, 4, 2, 0})
}

func TestConvertSemanticTokensToUTF16_PreservesRelativeEncodingAcrossTokens(t *testing.T) {
	// Two tokens on one line after an emoji: the second delta must be recomputed
	// against the converted position of the first, not the raw one.
	got := convertSemanticTokensToUTF16([]int{
		0, 4, 2, 1, 0, // "ab" at UTF-8 offset 4 -> UTF-16 offset 2
		0, 3, 2, 2, 0, // "cd" at UTF-8 offset 7 -> UTF-16 offset 5
	}, "😀ab cd", PositionEncodingUTF8)

	assertTokenData(t, got, []int{
		0, 2, 2, 1, 0,
		0, 3, 2, 2, 0,
	})
}

func TestConvertSemanticTokensToUTF16_SkipsOutOfRangeLines(t *testing.T) {
	got := convertSemanticTokensToUTF16([]int{9, 0, 2, 1, 0}, "only one line", PositionEncodingUTF8)
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestSemanticTokensWire_ParsesRelativeData(t *testing.T) {
	var payload semanticTokensWire
	if err := json.Unmarshal(json.RawMessage(`{"data":[0,0,4,1,0,1,2,3,3,5]}`), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(payload.Data) != 10 {
		t.Fatalf("data length = %d, want 10", len(payload.Data))
	}
}

// The Monaco provider registers from the legend, so an empty token response must
// still carry it or semantic highlighting never starts for that document.
func TestRequestSemanticTokens_KeepsLegendWhenServerHasNoTokensYet(t *testing.T) {
	legend := SemanticTokensLegend{TokenTypes: []string{"variable"}, TokenModifiers: []string{"readonly"}}

	for _, body := range []string{`null`, `{"data":[]}`} {
		t.Run(body, func(t *testing.T) {
			clientToServerR, clientToServerW := io.Pipe()
			serverToClientR, serverToClientW := io.Pipe()
			defer func() {
				_ = clientToServerR.Close()
				_ = clientToServerW.Close()
				_ = serverToClientR.Close()
				_ = serverToClientW.Close()
			}()

			transport := NewTransport(clientToServerW, serverToClientR)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				_ = transport.ReadLoop(ctx)
			}()

			done := make(chan error, 1)
			go func() {
				reader := bufio.NewReader(clientToServerR)
				msg, err := ReadMessage(reader)
				if err != nil {
					done <- err
					return
				}
				resp := Message{JSONRPC: "2.0", ID: msg.ID, Result: json.RawMessage(body)}
				done <- WriteMessage(serverToClientW, resp)
			}()

			got, err := RequestSemanticTokens(ctx, transport, "file:///main.go", "x := 1\n", PositionEncodingUTF16, legend)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("result = nil, want the legend preserved")
			}
			if len(got.Legend.TokenTypes) != 1 || got.Legend.TokenTypes[0] != "variable" {
				t.Fatalf("legend = %+v", got.Legend)
			}
			if len(got.Data) != 0 {
				t.Fatalf("data = %v, want empty", got.Data)
			}

			if err := <-done; err != nil {
				t.Fatalf("server flow failed: %v", err)
			}
		})
	}
}

func TestRequestSemanticTokens_ReturnsNilWhenServerHasNoLegend(t *testing.T) {
	got, err := RequestSemanticTokens(context.Background(), nil, "file:///main.go", "", PositionEncodingUTF16, SemanticTokensLegend{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("result = %+v, want nil when the server advertises no legend", got)
	}
}

func assertTokenData(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
