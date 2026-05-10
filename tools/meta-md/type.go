package metamd

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
)

func fileType(filename string) string {
	if runtime.GOOS == "windows" {
		return ""
	}

	cmd := exec.Command("file", filename)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return ""
	}

	return strings.ReplaceAll(out.String()[len(filename)+2:], "`", "'")
}
