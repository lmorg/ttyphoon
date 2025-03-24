package getshell

import (
	"runtime"
	"testing"
)

func TestDscl(t *testing.T) {
	if runtime.GOOS != _OS_MACOS {
		t.Skip("test for macOS")
		return
	}

	shell, err := dscl()
	if err != nil {
		t.Error(err)
	}

	t.Log(shell)
}
