package printing

import "context"

func Print(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return printWindow(ctx)
}
