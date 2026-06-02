package notes

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

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

	projectFiles := make([]string, 0, 128)

	err := filepath.WalkDir(ulf.PathProjectRoot, func(path string, d os.DirEntry, err error) error {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		if err != nil {
			log.Println(err)
			return nil
		}
		if d.IsDir() {
			if len(d.Name()) == 0 || d.Name()[0] == '.' || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if len(d.Name()) == 0 || d.Name()[0] == '.' {
			return nil
		}

		if isSystemFileName(d.Name()) {
			return nil
		}

		filename := strings.Replace(path, ulf.PathProjectRoot, "$PROJECT", 1)
		projectFiles = append(projectFiles, filename)

		return nil
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return ulf
		}
		log.Println(err)
	}

	ulf.Files = append(ulf.Files, projectFiles...)
	return ulf
}
