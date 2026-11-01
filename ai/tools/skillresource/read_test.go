package skillresource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSkillResourceListsAndReadsRelativeFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "references"), 0700); err != nil {
		t.Fatal(err)
	}
	resource := filepath.Join(root, "references", "guide.md")
	if err := os.WriteFile(resource, []byte("first\nsecond\nthird\n"), 0600); err != nil {
		t.Fatal(err)
	}
	directories := map[string]string{"review": root}

	listing, err := readSkillResource(directories, "review", "references", nil, nil)
	if err != nil || !strings.Contains(listing, "references/guide.md") {
		t.Fatalf("directory listing = %q, %v", listing, err)
	}
	resolvedResource, err := filepath.EvalSymlinks(resource)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := readSkillResource(directories, "review", "references/guide.md", intPointer(2), intPointer(2))
	if err != nil || !strings.Contains(contents, "Absolute path: "+resolvedResource) || !strings.HasSuffix(contents, "second") {
		t.Fatalf("ranged resource = %q, %v", contents, err)
	}
}

func TestReadSkillResourceRejectsInactiveAndEscapingPaths(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSkillResource(nil, "review", "references/guide.md", nil, nil); err == nil {
		t.Fatal("inactive skill resource was readable")
	}
	directories := map[string]string{"review": root}
	for _, path := range []string{"../outside.md", outside} {
		if _, err := readSkillResource(directories, "review", path, nil, nil); err == nil {
			t.Errorf("escaping path %q was readable", path)
		}
	}
	link := filepath.Join(root, "outside.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := readSkillResource(directories, "review", "outside.md", nil, nil); err == nil {
		t.Fatal("symlink escaping the skill directory was readable")
	}
}

func TestReadSkillResourceRejectsBinaryAndOversizedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte{'a', 0, 'b'}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.txt"), make([]byte, maxSkillResourceBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	directories := map[string]string{"review": root}
	for _, path := range []string{"binary.dat", "large.txt"} {
		if _, err := readSkillResource(directories, "review", path, nil, nil); err == nil {
			t.Errorf("unsupported resource %q was read", path)
		}
	}
}

func intPointer(value int) *int { return &value }
