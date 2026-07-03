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
	"github.com/lmorg/ttyphoon/utils/cache"
)

type ListFilesReturnT struct {
	PathProjectRoot string
	PathGlobalNotes string
	PathUserNotes   string
	GroupName       string // tmux window name (tab name)
	Files           []string
}

const MAX_FILES = 5_000

func ListFiles(ctx context.Context, renderer types.Renderer) *ListFilesReturnT {
	ulf := &ListFilesReturnT{}

	tile := renderer.ActiveTile()
	if tile == nil {
		return ulf
	}

	ulf.PathProjectRoot = DirProjectRoot(tile.Pwd())
	ulf.PathGlobalNotes = DirGlobal()
	ulf.GroupName = tile.GroupName()
	ulf.PathUserNotes = ulf.PathGlobalNotes + ulf.GroupName + "/"

	cache.Read(cache.NS_NOTESW_FILES, ulf.PathUserNotes, &ulf.Files)
	cache.Write(cache.NS_NOTESW_FILES, ulf.PathUserNotes, &ulf.Files, cache.Days(365))

	ulf.Files = append(ulf.Files, listFiles(ulf.PathGlobalNotes, "GLOBAL")...)
	ulf.Files = append(ulf.Files, listFiles(ulf.PathUserNotes, "NOTES")...)
	//files = append(files, listFiles(a.historyDir, "HISTORY")...)

	if ulf.PathProjectRoot == "" {
		return ulf
	}

	var (
		projectFiles []string
		count        int
	)

	err := filepath.WalkDir(ulf.PathProjectRoot, func(path string, d os.DirEntry, err error) error {
		count++

		if count > MAX_FILES {
			return fmt.Errorf(`filesystem walk: too many files. MAX_FILES=%d`, MAX_FILES)
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

	ulf.Files = append(ulf.Files, projectFiles...)

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Println(err)
	}

	return ulf
}
