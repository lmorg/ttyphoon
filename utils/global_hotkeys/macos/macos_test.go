//go:build darwin
// +build darwin

package macos

import "testing"

const (
	ModCommand uint32 = 0x0100 // cmdKey (⌘)
	ModShift   uint32 = 0x0200 // shiftKey (⇧)
	KeyF12     uint32 = 0x6F
)

func TestRegisterUnregister(t *testing.T) {
	hk, err := Register(KeyF12, ModCommand|ModShift, func() {})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if hk == nil || hk.ref == 0 {
		t.Fatalf("expected a valid hotkey ref")
	}

	registry.mutex.RLock()
	got := registry.byID[hk.id]
	registry.mutex.RUnlock()
	if got != hk {
		t.Fatalf("hotkey not present in registry")
	}

	if err := hk.Unregister(); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	registry.mutex.RLock()
	_, exists := registry.byID[hk.id]
	registry.mutex.RUnlock()
	if exists {
		t.Fatalf("hotkey still present after Unregister")
	}

	// Second Unregister must be a no-op.
	if err := hk.Unregister(); err != nil {
		t.Fatalf("second Unregister returned error: %v", err)
	}
}
