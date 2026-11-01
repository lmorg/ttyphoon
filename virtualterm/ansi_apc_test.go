package virtualterm

import (
	"encoding/base64"
	"reflect"
	"testing"
)

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
