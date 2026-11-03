//go:build darwin && cgo

package printing

/*
#cgo CFLAGS: -fblocks
#cgo LDFLAGS: -framework Cocoa -framework WebKit
int ttyphoon_print_window(void);
*/
import "C"

import (
	"context"
	"fmt"
)

func printWindow(_ context.Context) error {
	switch result := int(C.ttyphoon_print_window()); result {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("macOS printing requires an active application webview")
	case 2:
		return fmt.Errorf("native webview printing requires macOS 11 or later")
	default:
		return fmt.Errorf("could not start the native macOS print operation (code %d)", result)
	}
}
