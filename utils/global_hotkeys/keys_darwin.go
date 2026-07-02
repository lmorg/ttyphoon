//go:build darwin
// +build darwin

package globalhotkeys

// Carbon modifier masks accepted by Register. They may be combined with bitwise
// OR, e.g. ModCommand | ModShift.
//
// These are the classic Carbon modifier bit masks (from <Carbon/Events.h>), not
// the Cocoa NSEventModifierFlags.
const (
	ModCommand uint32 = 0x0100 // cmdKey (⌘)
	ModShift   uint32 = 0x0200 // shiftKey (⇧)
	ModOption  uint32 = 0x0800 // optionKey (⌥)
	ModControl uint32 = 0x1000 // controlKey (⌃)
)

// Virtual key codes (from <Carbon/Events.h>, kVK_* constants). These are layout
// independent positional codes, as required by RegisterEventHotKey.
const (
	KeyA uint32 = 0x00
	KeyS uint32 = 0x01
	KeyD uint32 = 0x02
	KeyF uint32 = 0x03
	KeyH uint32 = 0x04
	KeyG uint32 = 0x05
	KeyZ uint32 = 0x06
	KeyX uint32 = 0x07
	KeyC uint32 = 0x08
	KeyV uint32 = 0x09
	KeyB uint32 = 0x0B
	KeyQ uint32 = 0x0C
	KeyW uint32 = 0x0D
	KeyE uint32 = 0x0E
	KeyR uint32 = 0x0F
	KeyY uint32 = 0x10
	KeyT uint32 = 0x11
	KeyO uint32 = 0x1F
	KeyU uint32 = 0x20
	KeyI uint32 = 0x22
	KeyP uint32 = 0x23
	KeyL uint32 = 0x25
	KeyJ uint32 = 0x26
	KeyK uint32 = 0x28
	KeyN uint32 = 0x2D
	KeyM uint32 = 0x2E

	Key1 uint32 = 0x12
	Key2 uint32 = 0x13
	Key3 uint32 = 0x14
	Key4 uint32 = 0x15
	Key5 uint32 = 0x17
	Key6 uint32 = 0x16
	Key7 uint32 = 0x1A
	Key8 uint32 = 0x1C
	Key9 uint32 = 0x19
	Key0 uint32 = 0x1D

	KeyReturn     uint32 = 0x24
	KeyTab        uint32 = 0x30
	KeySpace      uint32 = 0x31
	KeyDelete     uint32 = 0x33
	KeyEscape     uint32 = 0x35
	KeyLeftArrow  uint32 = 0x7B
	KeyRightArrow uint32 = 0x7C
	KeyDownArrow  uint32 = 0x7D
	KeyUpArrow    uint32 = 0x7E

	KeyF1  uint32 = 0x7A
	KeyF2  uint32 = 0x78
	KeyF3  uint32 = 0x63
	KeyF4  uint32 = 0x76
	KeyF5  uint32 = 0x60
	KeyF6  uint32 = 0x61
	KeyF7  uint32 = 0x62
	KeyF8  uint32 = 0x64
	KeyF9  uint32 = 0x65
	KeyF10 uint32 = 0x6D
	KeyF11 uint32 = 0x67
	KeyF12 uint32 = 0x6F
)
