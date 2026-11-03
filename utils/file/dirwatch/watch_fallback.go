//go:build !linux && (!darwin || !cgo)

package dirwatch

import (
	"context"
	"errors"
)

func startNative(context.Context, string, func(string) bool, bool) (nativeWatcher, error) {
	return nil, errors.New("native directory watching requires Linux or macOS with cgo")
}
