//go:build darwin && !cgo

package printing

import (
	"context"
	"fmt"
)

func printWindow(_ context.Context) error {
	return fmt.Errorf("native macOS printing requires cgo")
}
