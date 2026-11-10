package virtualterm

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestMxCommandTransports(t *testing.T) {
	for _, transport := range []struct {
		name   string
		prefix string
	}{
		{name: "APC", prefix: "\x1b_"},
		{name: "OSC1701", prefix: "\x1b]1701;"},
	} {
		for _, terminator := range []struct {
			name  string
			value string
		}{
			{name: "ST", value: _TERMINATE_ST},
			{name: "BEL", value: _TERMINATE_BELL},
		} {
			t.Run(transport.name+"/"+terminator.name, func(t *testing.T) {
				t.Run("completion", func(t *testing.T) {
					term := newParserTestTerm()
					pty := term.Pty.(*parserTestPty)
					stream := transport.prefix + `ttyphoon;interactive-action-complete;{"id":"one;two;\u03bb","exitCode":7}` + terminator.value + "X"
					pty.FeedInput([]byte(stream))
					drainMockPtyInput(t, term, len(stream)*2)

					want := interactiveActionCompletion{ID: "one;two;\u03bb", ExitCode: 7}
					if term.pendingInteractiveAction == nil || *term.pendingInteractiveAction != want {
						t.Fatalf("completion = %+v, want %+v", term.pendingInteractiveAction, want)
					}
					if got := (*term.screen)[0].Cells[0].Char; got != 'X' {
						t.Fatalf("character after terminator = %q, want X", got)
					}
				})

				t.Run("output-block", func(t *testing.T) {
					term := newParserTestTerm()
					pty := term.Pty.(*parserTestPty)
					stream := transport.prefix + `begin;output-block;{"CmdLine":"echo one; echo \u03bb","EnvVars":{"VALUE":"one;two"}}` + terminator.value
					pty.FeedInput([]byte(stream))
					drainMockPtyInput(t, term, len(stream)*2)

					if got := string(term._blockMeta.Query); got != "echo one; echo \u03bb" {
						t.Fatalf("command = %q", got)
					}
					if got := term._blockMeta.EnvVars["VALUE"]; got != "one;two" {
						t.Fatalf("environment VALUE = %q", got)
					}
				})
			})
		}
	}
}

type mxCommandTestRenderer struct {
	synchronizedOutputTestRenderer
	notifications []string
}

func (renderer *mxCommandTestRenderer) DisplayNotification(_ types.NotificationType, message string) {
	renderer.notifications = append(renderer.notifications, message)
}

func TestMxCommandEmptyAndUnknownPayloads(t *testing.T) {
	for _, prefix := range []string{"\x1b_", "\x1b]1701;", "\x1b]1701"} {
		term := newParserTestTerm()
		renderer := &mxCommandTestRenderer{}
		term.renderer = renderer
		pty := term.Pty.(*parserTestPty)
		stream := prefix + _TERMINATE_ST
		pty.FeedInput([]byte(stream))
		drainMockPtyInput(t, term, len(stream)*2)
		if len(renderer.notifications) != 1 {
			t.Fatalf("prefix %q: notifications = %v, want one diagnostic", prefix, renderer.notifications)
		}
	}

	for _, payload := range []string{"unknown", "begin;unknown", "end;unknown", "insert;unknown", "display;unknown", "ai;unknown", "config;unknown"} {
		term := newParserTestTerm()
		renderer := &mxCommandTestRenderer{}
		term.renderer = renderer
		pty := term.Pty.(*parserTestPty)
		stream := "\x1b]1701;" + payload + _TERMINATE_BELL
		pty.FeedInput([]byte(stream))
		drainMockPtyInput(t, term, len(stream)*2)
		if len(renderer.notifications) != 1 || !strings.HasSuffix(renderer.notifications[0], ": "+payload) {
			t.Fatalf("payload %q: notifications = %v, want complete payload diagnostic", payload, renderer.notifications)
		}
	}
}

func TestParseInteractiveActionCompletion(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	completion, err := parseInteractiveActionCompletion(`{"id":"` + id + `","exitCode":7}`)
	if err != nil {
		t.Fatal(err)
	}
	if completion.ID != id || completion.ExitCode != 7 {
		t.Fatalf("completion = %+v, want id %q and exit code 7", completion, id)
	}
}

func TestDecodeOutputBlockSnapshot(t *testing.T) {
	command := "git commit -m 'message'"
	environment := []byte("PATH=/bin:/usr/bin\x00CUSTOM_ENV=custom value\x00MULTILINE=one\ntwo\x00EMPTY=\x00")
	gotCommand, gotEnvironment, err := decodeOutputBlockSnapshot(
		base64.StdEncoding.EncodeToString([]byte(command)),
		base64.StdEncoding.EncodeToString(environment),
	)
	if err != nil {
		t.Fatal(err)
	}
	wantEnvironment := map[string]string{
		"PATH":       "/bin:/usr/bin",
		"CUSTOM_ENV": "custom value",
		"MULTILINE":  "one\ntwo",
		"EMPTY":      "",
	}
	if gotCommand != command || !reflect.DeepEqual(gotEnvironment, wantEnvironment) {
		t.Fatalf("snapshot = (%q, %#v), want (%q, %#v)", gotCommand, gotEnvironment, command, wantEnvironment)
	}
}

func TestParseInteractiveActionCompletionRejectsMissingOrInvalidExitCode(t *testing.T) {
	for _, payload := range []string{
		`{"id":"550e8400-e29b-41d4-a716-446655440000"}`,
		`{"id":"550e8400-e29b-41d4-a716-446655440000","exitCode":256}`,
		`{"id":"","exitCode":0}`,
	} {
		if _, err := parseInteractiveActionCompletion(payload); err == nil {
			t.Errorf("parseInteractiveActionCompletion(%q) unexpectedly succeeded", payload)
		}
	}
}
