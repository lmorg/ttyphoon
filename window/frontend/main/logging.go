package mainwindow

import (
	"context"
	"log"
	"os"
	"regexp"
	"strings"

	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

func (a *WApp) Log(s string) {
	log.Println(s)
}

// logWriter captures log output and sends it to the frontend
type logWriter struct {
	ctx context.Context
}

var rxLogCategory = regexp.MustCompile(`^\[[a-z]+\] `)

func (lw *logWriter) Write(p []byte) (int, error) {
	// Send each line to frontend, also write to original stderr
	text := string(p)
	if !rxLogCategory.MatchString(text[20:]) {
		text = text[:20] + "[debug] " + text[20:]
	}
	text = strings.ReplaceAll(strings.TrimSpace(text), "\n", " | ")

	if lw.ctx != nil {
		runtime.EventsEmit(lw.ctx, "notesLog", text)
	}
	// Also output to stderr so it's visible in console
	os.Stderr.Write(p) //nolint:errcheck
	return len(p), nil
}
