//go:build !darwin

package printing

import (
	"context"

	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

func printWindow(ctx context.Context) error {
	runtime.WindowPrint(ctx)
	return nil
}
