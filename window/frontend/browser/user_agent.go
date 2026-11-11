package browser

import (
	"context"
	_ "embed"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed safari.ver
var safariVersion string

var browserUserAgent = sync.OnceValues(func() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "plutil", "-extract", "CFBundleShortVersionString", "raw", "/Applications/Safari.app/Contents/Info.plist").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("browser user agent: uname -r: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return userAgentFromRelease(string(output))
})

func InitializeUserAgent() error {
	_, err := browserUserAgent()
	return err
}

func userAgentFromRelease(output string) (string, error) {
	release := strings.TrimSpace(output)
	if release == "" || strings.ContainsAny(release, " \t\r\n") {
		return "", fmt.Errorf("browser user agent: uname `plutil` output %q", output)
	}
	return fmt.Sprintf("Version/%s Safari/%s", release, safariVersion), nil
}
