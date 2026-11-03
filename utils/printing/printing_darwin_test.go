//go:build darwin && cgo

package printing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrintWithoutApplicationReportsUnavailableWebview(t *testing.T) {
	err := Print(context.Background())
	if err == nil || !strings.Contains(err.Error(), "active application webview") {
		t.Fatalf("Print() = %v, want unavailable webview", err)
	}
}

func TestNativePrintPDFContainsPreparedDocument(t *testing.T) {
	if os.Getenv("TTYPHOON_TEST_NATIVE_PRINT") != "1" {
		t.Skip("requires a macOS WindowServer session; enable TTYPHOON_TEST_NATIVE_PRINT=1")
	}
	directory := t.TempDir()
	binary := filepath.Join(directory, "print-pdf")
	command := exec.Command("clang", "-fblocks", "-framework", "Cocoa", "-framework", "WebKit", "-framework", "PDFKit", "testdata/print_pdf.m", "-o", binary)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile native fixture: %v\n%s", err, output)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	command = exec.CommandContext(ctx, binary, "../../frontend/src/notes.css", filepath.Join(directory, "printed.pdf"))
	command.WaitDelay = time.Second
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("native PDF output: %v\n%s", err, output)
	} else {
		t.Logf("native PDF output:\n%s", output)
	}
}
