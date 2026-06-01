package notes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirProjectRoot(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T) (cwd, home string, cleanup func())
		cwd            string
		expectedResult string
		description    string
	}{
		{
			name: "finds git root in current directory",
			setup: func(t *testing.T) (string, string, func()) {
				tmpdir := t.TempDir()
				gitDir := filepath.Join(tmpdir, ".git")
				if err := os.Mkdir(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				return tmpdir, "", func() {}
			},
			cwd:            "", // Will use the setup cwd
			expectedResult: "same-as-cwd",
			description:    "Should return cwd when .git exists in cwd",
		},
		{
			name: "finds git root one level up",
			setup: func(t *testing.T) (string, string, func()) {
				tmpdir := t.TempDir()
				gitDir := filepath.Join(tmpdir, ".git")
				if err := os.Mkdir(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				subdir := filepath.Join(tmpdir, "subdir")
				if err := os.Mkdir(subdir, 0755); err != nil {
					t.Fatal(err)
				}
				return subdir, "", func() {}
			},
			cwd:            "", // Will use the subdir
			expectedResult: "parent-of-cwd",
			description:    "Should return parent directory when .git exists there",
		},
		{
			name: "finds git root multiple levels up",
			setup: func(t *testing.T) (string, string, func()) {
				tmpdir := t.TempDir()
				gitDir := filepath.Join(tmpdir, ".git")
				if err := os.Mkdir(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				nested := filepath.Join(tmpdir, "a", "b", "c")
				if err := os.MkdirAll(nested, 0755); err != nil {
					t.Fatal(err)
				}
				return nested, "", func() {}
			},
			cwd:            "",
			expectedResult: "git-root",
			description:    "Should find .git directory multiple levels up",
		},
		{
			name: "returns cwd when no git found before reaching home",
			setup: func(t *testing.T) (string, string, func()) {
				tmpdir := t.TempDir()
				subdir := filepath.Join(tmpdir, "project")
				if err := os.Mkdir(subdir, 0755); err != nil {
					t.Fatal(err)
				}
				// tmpdir acts as home, subdir as cwd
				return subdir, tmpdir, func() {}
			},
			cwd:            "",
			expectedResult: "original-cwd",
			description:    "Should return original cwd when home directory is reached without finding .git",
		},
		{
			name: "uses provided cwd instead of os.Getwd",
			setup: func(t *testing.T) (string, string, func()) {
				tmpdir := t.TempDir()
				gitDir := filepath.Join(tmpdir, ".git")
				if err := os.Mkdir(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				return tmpdir, "", func() {}
			},
			cwd:            "use-setup",
			expectedResult: "same-as-cwd",
			description:    "Should use provided cwd parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cwd, home, cleanup := tt.setup(t)
			defer cleanup()

			// Save original home and override if test provided one
			originalHome := os.Getenv("HOME")
			if home != "" {
				os.Setenv("HOME", home)
				defer os.Setenv("HOME", originalHome)
			}

			// Use provided cwd or the one from setup
			testCwd := tt.cwd
			if testCwd == "" || testCwd == "use-setup" {
				testCwd = cwd
			}

			result := DirProjectRoot(testCwd)

			// Determine expected result
			var expected string
			switch tt.expectedResult {
			case "same-as-cwd":
				expected = testCwd
			case "parent-of-cwd":
				expected = filepath.Dir(testCwd)
			case "git-root":
				expected = cwd
				// Need to find where .git actually is
				current := testCwd
				for {
					if hasGitDirectory(current) {
						expected = current
						break
					}
					parent := filepath.Dir(current)
					if parent == current {
						break
					}
					current = parent
				}
			case "original-cwd":
				expected = testCwd
			default:
				t.Fatalf("unknown expected result: %s", tt.expectedResult)
			}

			if result != expected {
				t.Errorf("%s: got %q, expected %q", tt.description, result, expected)
			}
		})
	}
}

func TestHasGitDirectory(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "returns true when .git exists",
			setup: func(t *testing.T) string {
				tmpdir := t.TempDir()
				gitDir := filepath.Join(tmpdir, ".git")
				if err := os.Mkdir(gitDir, 0755); err != nil {
					t.Fatal(err)
				}
				return tmpdir
			},
			expected: true,
		},
		{
			name: "returns false when .git does not exist",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: false,
		},
		{
			name: "returns false when .git is a file not a directory",
			setup: func(t *testing.T) string {
				tmpdir := t.TempDir()
				gitFile := filepath.Join(tmpdir, ".git")
				if err := os.WriteFile(gitFile, []byte(""), 0644); err != nil {
					t.Fatal(err)
				}
				return tmpdir
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			result := hasGitDirectory(path)
			if result != tt.expected {
				t.Errorf("got %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDirProjectRootEdgeCases(t *testing.T) {
	t.Run("empty cwd and os.Getwd fails", func(t *testing.T) {
		// This test verifies the error handling when cwd is empty
		// and os.Getwd() would fail (we can't easily mock os.Getwd,
		// but we can test that empty string is handled)
		result := DirProjectRoot("")
		// Result depends on current working directory, just verify it's not a panic
		_ = result
	})

	t.Run("explicit cwd with git in multiple levels", func(t *testing.T) {
		tmpdir := t.TempDir()
		gitDir := filepath.Join(tmpdir, ".git")
		if err := os.Mkdir(gitDir, 0755); err != nil {
			t.Fatal(err)
		}

		deepPath := filepath.Join(tmpdir, "a", "b", "c", "d")
		if err := os.MkdirAll(deepPath, 0755); err != nil {
			t.Fatal(err)
		}

		result := DirProjectRoot(deepPath)
		if result != tmpdir {
			t.Errorf("expected %q, got %q", tmpdir, result)
		}
	})

	t.Run("git directory at search root", func(t *testing.T) {
		tmpdir := t.TempDir()
		gitDir := filepath.Join(tmpdir, ".git")
		if err := os.Mkdir(gitDir, 0755); err != nil {
			t.Fatal(err)
		}

		result := DirProjectRoot(tmpdir)
		if result != tmpdir {
			t.Errorf("expected %q, got %q", tmpdir, result)
		}
	})
}
