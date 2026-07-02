//go:build darwin
// +build darwin

// Package macos registers system-wide ("global") hotkeys on macOS using the
// idiomatic Carbon Event Manager APIs (RegisterEventHotKey / InstallEventHandler).
//
// The Carbon framework is loaded and called dynamically via PureGo
// (github.com/ebitengine/purego) so the package builds without cgo.
//
// Global hotkeys are delivered through the application's main run loop. In a
// Cocoa application (such as one hosted by Wails) the Carbon hotkey events are
// dispatched automatically once the NSApplication run loop is running; no extra
// run loop needs to be pumped by this package.
package macos

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Carbon event constants.
const (
	kEventClassKeyboard     = 0x6B657962 // 'keyb'
	kEventHotKeyPressed     = 5
	kEventParamDirectObject = 0x2D2D2D2D // '----'
	typeEventHotKeyID       = 0x686B6964 // 'hkid'

	// hotkeySignature is the four-char-code identifying hotkeys owned by this
	// package ('typh').
	hotkeySignature = 0x74797068

	noErr = 0
)

// eventHotKeyID mirrors the Carbon EventHotKeyID struct.
type eventHotKeyID struct {
	signature uint32
	id        uint32
}

// eventTypeSpec mirrors the Carbon EventTypeSpec struct.
type eventTypeSpec struct {
	eventClass uint32
	eventKind  uint32
}

// carbon holds the dynamically resolved Carbon symbols.
var carbon struct {
	once sync.Once
	err  error

	registerEventHotKey       uintptr
	unregisterEventHotKey     uintptr
	getApplicationEventTarget uintptr
	installEventHandler       uintptr
	getEventParameter         uintptr
}

// loadCarbon dynamically links the Carbon framework and installs the shared
// Carbon event handler used to route every hotkey press. It is safe to call
// repeatedly; the work is performed exactly once.
func loadCarbon() error {
	carbon.once.Do(func() {
		handle, err := purego.Dlopen(
			"/System/Library/Frameworks/Carbon.framework/Carbon",
			purego.RTLD_LAZY|purego.RTLD_GLOBAL,
		)
		if err != nil {
			carbon.err = fmt.Errorf("unable to load Carbon framework: %w", err)
			return
		}

		symbols := []struct {
			name string
			ptr  *uintptr
		}{
			{"RegisterEventHotKey", &carbon.registerEventHotKey},
			{"UnregisterEventHotKey", &carbon.unregisterEventHotKey},
			{"GetApplicationEventTarget", &carbon.getApplicationEventTarget},
			{"InstallEventHandler", &carbon.installEventHandler},
			{"GetEventParameter", &carbon.getEventParameter},
		}

		for _, s := range symbols {
			sym, err := purego.Dlsym(handle, s.name)
			if err != nil {
				carbon.err = fmt.Errorf("unable to resolve Carbon symbol %s: %w", s.name, err)
				return
			}
			*s.ptr = sym
		}

		carbon.err = installSharedHandler()
	})

	return carbon.err
}

// eventTypeList must stay alive for the lifetime of the installed handler, so
// it is held at package scope rather than on the stack.
var eventTypeList = eventTypeSpec{
	eventClass: kEventClassKeyboard,
	eventKind:  kEventHotKeyPressed,
}

// installSharedHandler installs a single Carbon event handler on the
// application event target. All hotkey presses are routed through it and
// dispatched by hotkey id.
func installSharedHandler() error {
	target, _, _ := purego.SyscallN(carbon.getApplicationEventTarget)
	if target == 0 {
		return fmt.Errorf("GetApplicationEventTarget returned NULL")
	}

	callback := purego.NewCallback(hotkeyEventHandler)

	var handlerRef uintptr
	status, _, _ := purego.SyscallN(
		carbon.installEventHandler,
		target,
		callback,
		1, // inNumTypes
		uintptr(unsafe.Pointer(&eventTypeList)),
		0, // inUserData
		uintptr(unsafe.Pointer(&handlerRef)),
	)
	if status != noErr {
		return fmt.Errorf("InstallEventHandler failed: OSStatus %d", int32(status))
	}

	go dispatcher()
	return nil
}

// hotkeyEventHandler is the C callback invoked by Carbon on the main run loop
// for every hotkey press. It must do as little work as possible, so it merely
// resolves the hotkey id and hands the registered callback to the dispatcher
// goroutine.
func hotkeyEventHandler(_ uintptr, theEvent uintptr, _ uintptr) uintptr {
	var hkID eventHotKeyID
	status, _, _ := purego.SyscallN(
		carbon.getEventParameter,
		theEvent,
		uintptr(kEventParamDirectObject),
		uintptr(typeEventHotKeyID),
		0, // outActualType (NULL)
		unsafe.Sizeof(hkID),
		0, // outActualSize (NULL)
		uintptr(unsafe.Pointer(&hkID)),
	)
	if status != noErr {
		return noErr
	}

	registry.mutex.RLock()
	hk := registry.byID[hkID.id]
	registry.mutex.RUnlock()

	if hk != nil {
		select {
		case dispatchCh <- hk.callback:
		default:
		}
	}

	return noErr
}

// dispatchCh carries registered callbacks from the Carbon handler (running on
// the main thread) to the dispatcher goroutine, keeping the run loop responsive.
var dispatchCh = make(chan func(), 32)

func dispatcher() {
	for fn := range dispatchCh {
		if fn != nil {
			fn()
		}
	}
}

// registry tracks every live hotkey, keyed by its unique id.
var registry = struct {
	mutex  sync.RWMutex
	byID   map[uint32]*Hotkey
	nextID uint32
}{
	byID: make(map[uint32]*Hotkey),
}

// Hotkey represents a single registered global hotkey. Call Unregister to
// release it.
type Hotkey struct {
	id       uint32
	ref      uintptr // EventHotKeyRef
	callback func()
}

// Register binds a system-wide hotkey for the given virtual key code and
// modifier mask, invoking callback whenever it is pressed. Use the Key* and
// Mod* constants for keyCode and modifiers respectively. Modifiers may be
// combined with bitwise OR.
//
// The callback runs on a dedicated goroutine, not the main run loop, so it is
// safe for it to perform blocking work.
func Register(keyCode, modifiers uint32, callback func()) (*Hotkey, error) {
	if callback == nil {
		return nil, fmt.Errorf("hotkey callback must not be nil")
	}

	if err := loadCarbon(); err != nil {
		return nil, err
	}

	registry.mutex.Lock()
	registry.nextID++
	id := registry.nextID
	registry.mutex.Unlock()

	target, _, _ := purego.SyscallN(carbon.getApplicationEventTarget)
	if target == 0 {
		return nil, fmt.Errorf("GetApplicationEventTarget returned NULL")
	}

	// EventHotKeyID is an 8-byte, all-integer struct passed by value. On both
	// arm64 and amd64 it is passed in a single general-purpose register, so it
	// is packed into one machine word: low 32 bits = signature, high = id.
	packedID := uintptr(uint64(hotkeySignature) | uint64(id)<<32)

	var ref uintptr
	status, _, _ := purego.SyscallN(
		carbon.registerEventHotKey,
		uintptr(keyCode),
		uintptr(modifiers),
		packedID,
		target,
		0, // inOptions
		uintptr(unsafe.Pointer(&ref)),
	)
	if status != noErr {
		return nil, fmt.Errorf("RegisterEventHotKey failed: OSStatus %d", int32(status))
	}

	hk := &Hotkey{id: id, ref: ref, callback: callback}

	registry.mutex.Lock()
	registry.byID[id] = hk
	registry.mutex.Unlock()

	return hk, nil
}

// Unregister releases the hotkey. It is safe to call more than once.
func (hk *Hotkey) Unregister() error {
	if hk == nil || hk.ref == 0 {
		return nil
	}

	status, _, _ := purego.SyscallN(carbon.unregisterEventHotKey, hk.ref)

	registry.mutex.Lock()
	delete(registry.byID, hk.id)
	registry.mutex.Unlock()

	hk.ref = 0

	if status != noErr {
		return fmt.Errorf("UnregisterEventHotKey failed: OSStatus %d", int32(status))
	}
	return nil
}
