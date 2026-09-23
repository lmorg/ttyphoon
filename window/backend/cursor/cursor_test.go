package cursor

import (
	"reflect"
	"testing"
)

func TestBaseCursorWaitsForTemporaryOverrideToEnd(t *testing.T) {
	var emitted []string
	Register(func(css string) {
		emitted = append(emitted, css)
	})
	t.Cleanup(func() { Register(nil) })

	SetBase("text")
	Hand()
	SetBase("default")

	if expected := []string{"text", "pointer"}; !reflect.DeepEqual(emitted, expected) {
		t.Fatalf("expected active override to remain visible, got %v", emitted)
	}

	Arrow()
	if expected := []string{"text", "pointer", "default"}; !reflect.DeepEqual(emitted, expected) {
		t.Fatalf("expected hover exit to restore the latest base cursor, got %v", emitted)
	}
}

func TestArrowRestoresPointerBaseWithoutDuplicateEmission(t *testing.T) {
	var emitted []string
	Register(func(css string) {
		emitted = append(emitted, css)
	})
	t.Cleanup(func() { Register(nil) })

	SetBase("pointer")
	Hand()
	Arrow()

	if expected := []string{"pointer"}; !reflect.DeepEqual(emitted, expected) {
		t.Fatalf("expected restoring an identical base cursor not to emit twice, got %v", emitted)
	}
}

func TestActivateBaseClearsPreviousPaneOverride(t *testing.T) {
	var emitted []string
	Register(func(css string) {
		emitted = append(emitted, css)
	})
	t.Cleanup(func() { Register(nil) })

	Hand()
	ActivateBase("text")

	if expected := []string{"pointer", "text"}; !reflect.DeepEqual(emitted, expected) {
		t.Fatalf("expected pane activation to replace the stale override, got %v", emitted)
	}
}
