package file

import (
	"path/filepath"
	"testing"
)

func TestDocumentsDir(t *testing.T) {
	for _, home := range []string{"/home/user", "/home/user with spaces", "/home/user/"} {
		if got, want := DocumentsDir(home), filepath.Join(home, "Documents", "ttyphoon"); got != want {
			t.Errorf("DocumentsDir(%q) = %q, want %q", home, got, want)
		}
	}
}
