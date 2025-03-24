package getshell

import (
	"runtime"
	"testing"
)

func TestGetent(t *testing.T) {
	if runtime.GOOS != _OS_LINUX {
		t.Skip("test for Linux")
		return
	}

	shell, err := getent()
	if err != nil {
		t.Error(err)
	}

	t.Log(shell)
}
