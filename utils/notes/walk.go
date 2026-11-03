package notes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

type FileContext struct {
	PathProjectRoot string `json:"projectRoot"`
	PathGlobalNotes string `json:"globalNotes"`
	PathUserNotes   string `json:"userNotes"`
	GroupName       string `json:"groupName"`
}

type ListFilesReturnT struct {
	PathProjectRoot string
	PathGlobalNotes string
	PathUserNotes   string
	GroupName       string // tmux window name (tab name)
	Files           []string
	Err             error
	Truncated       bool
	Entries         int
}

const MAX_FILES = 5_000

var errFileScanLimit = errors.New("filesystem entry limit reached")

func ListFiles(ctx context.Context, renderer types.Renderer) *ListFilesReturnT {
	tile := renderer.ActiveTile()
	return ScanFiles(ctx, ResolveFileContext(tile))
}

func ResolveFileContext(tile types.Tile) FileContext {
	if tile == nil {
		return FileContext{}
	}
	return ResolveDirectoryContext(tile.Pwd(), tile.GroupName())
}

func ResolveDirectoryContext(pwd, group string) FileContext {
	global := DirGlobal()
	projectRoot := ""
	if pwd != "" {
		projectRoot = DirProjectRoot(pwd)
	}
	return FileContext{
		PathProjectRoot: projectRoot,
		PathGlobalNotes: global,
		PathUserNotes:   global + group + "/",
		GroupName:       group,
	}
}

func FileContextRoots(roots FileContext) []string {
	paths := []string{}
	seen := make(map[string]bool)
	for _, path := range []string{roots.PathProjectRoot, roots.PathUserNotes, roots.PathGlobalNotes} {
		root := indexRoot(path)
		if root != "" && !seen[root] {
			paths = append(paths, root)
			seen[root] = true
		}
	}
	return paths
}

func ScanFiles(ctx context.Context, roots FileContext) *ListFilesReturnT {
	if ctx == nil {
		ctx = context.Background()
	}
	ulf := &ListFilesReturnT{
		PathProjectRoot: roots.PathProjectRoot,
		PathGlobalNotes: roots.PathGlobalNotes,
		PathUserNotes:   roots.PathUserNotes,
		GroupName:       roots.GroupName,
	}
	if err := ctx.Err(); err != nil {
		ulf.Err = err
		return ulf
	}

	ulf.Files = append(ulf.Files, listFiles(ulf.PathGlobalNotes, "GLOBAL")...)
	ulf.Files = append(ulf.Files, listFiles(ulf.PathUserNotes, "NOTES")...)
	//files = append(files, listFiles(a.historyDir, "HISTORY")...)

	if ulf.PathProjectRoot == "" {
		ulf.Err = ctx.Err()
		if ulf.Err != nil {
			ulf.Files = nil
		}
		return ulf
	}

	var (
		projectFiles []string
		count        int
		scanError    error
	)

	err := filepath.WalkDir(ulf.PathProjectRoot, func(path string, d os.DirEntry, err error) error {
		count++

		if count > MAX_FILES {
			return fmt.Errorf("%w: MAX_FILES=%d", errFileScanLimit, MAX_FILES)
		}

		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		if err != nil {
			log.Printf("filesystem walk: %v", err)
			scanError = err
			return nil
		}
		if d.IsDir() {
			if len(d.Name()) == 0 {
				return filepath.SkipDir
			}
			for dir, ignore := range config.Config.Notes.ExcludeDirectories {
				if ignore && dir == d.Name() {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if len(d.Name()) == 0 /*|| d.Name()[0] == '.'*/ {
			return nil
		}

		if isSystemFileName(d.Name()) {
			return nil
		}

		filename := strings.Replace(path, ulf.PathProjectRoot, "$PROJECT", 1)
		projectFiles = append(projectFiles, filename)

		return nil
	})

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		ulf.Err = ctx.Err()
		if ulf.Err == nil {
			ulf.Err = err
		}
		ulf.Files = nil
		return ulf
	}

	if errors.Is(err, errFileScanLimit) {
		ulf.Truncated = true
	} else if err != nil {
		ulf.Err = err
	}
	if scanError != nil {
		ulf.Err = scanError
	}
	if ulf.Err != nil {
		ulf.Files = nil
		return ulf
	}
	ulf.Entries = min(count, MAX_FILES)
	ulf.Files = append(ulf.Files, projectFiles...)

	return ulf
}
