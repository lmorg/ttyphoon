package notes

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/utils/cache"
)

type HistoryListT struct {
	History []HistoryEntryT
	Index   int
}

type HistoryEntryT struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}

func (entry *HistoryEntryT) UnmarshalJSON(data []byte) error {
	var filename string
	if err := json.Unmarshal(data, &filename); err == nil {
		entry.Filename = filename
		entry.Line = 1
		return nil
	}

	type historyEntryAlias HistoryEntryT
	var decoded historyEntryAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*entry = HistoryEntryT(decoded)
	if entry.Line < 1 {
		entry.Line = 1
	}
	return nil
}

var historyMigrationMu sync.Mutex

func MigrateWorkspaceHistory(workspace, projectRoot string) {
	historyMigrationMu.Lock()
	defer historyMigrationMu.Unlock()
	var history HistoryListT
	if cache.Read(cache.NS_NOTESW_HISTORY, "workspace:"+workspace, &history) {
		return
	}
	cache.Read(cache.NS_NOTESW_HISTORY, projectRoot, &history)
	setHistoryList(workspace, &history)
}

func getHistoryList(workspace string) *HistoryListT {
	h := &HistoryListT{}

	cache.Read(cache.NS_NOTESW_HISTORY, "workspace:"+workspace, h)

	return h
}

func setHistoryList(workspace string, h *HistoryListT) {
	cache.Write(cache.NS_NOTESW_HISTORY, "workspace:"+workspace, h, cache.Days(30))
}

func HistoryRecentFiles(workspace string) []string {
	history := getHistoryList(workspace)
	recent := make([]string, 0, 10)
	seen := make(map[string]bool)
	for index := len(history.History) - 1; index >= 0 && len(recent) < 10; index-- {
		filename := history.History[index].Filename
		if filename == "" || seen[filename] {
			continue
		}
		seen[filename] = true
		recent = append(recent, filename)
	}
	return recent
}

func HistoryListCurrent(workspace string) string {
	return HistoryListCurrentEntry(workspace).Filename
}

func HistoryListCurrentEntry(workspace string) HistoryEntryT {
	h := getHistoryList(workspace)
	if len(h.History) == 0 {
		return HistoryEntryT{}
	}
	if h.Index < 0 || h.Index >= len(h.History) {
		return HistoryEntryT{}
	}

	return h.History[h.Index]
}

func HistoryListAdd(workspace, filename string) error {
	return HistoryListAddAt(workspace, filename, 1)
}

func HistoryListAddAt(workspace, filename string, line int) error {
	if filename == "" {
		return errors.New("no filename to add to history")
	}

	filename = filepath.Clean(filename)
	if line < 1 {
		line = 1
	}

	h := getHistoryList(workspace)
	if h.Index >= 0 && h.Index < len(h.History) {
		current := h.History[h.Index]
		if filepath.Clean(current.Filename) == filename && abs(current.Line-line) < 20 {
			return nil
		}
	}

	truncateTo := h.Index + 1
	if truncateTo < 0 {
		truncateTo = 0
	}
	if truncateTo > len(h.History) {
		truncateTo = len(h.History)
	}

	h.History = h.History[:truncateTo]

	h.History = append(h.History, HistoryEntryT{Filename: filename, Line: line})
	if len(h.History) > 50 {
		h.History = h.History[len(h.History)-50:]
	}

	h.Index = len(h.History) - 1
	setHistoryList(workspace, h)
	return nil
}

func HistoryListRename(workspace, oldFile, newFile string) error {
	if oldFile == "" || newFile == "" || oldFile == newFile {
		return fmt.Errorf("cannot update history: from '%s' to '%s'", oldFile, newFile)
	}

	oldFile = filepath.Clean(oldFile)
	newFile = filepath.Clean(newFile)

	h := getHistoryList(workspace)

	for i := range h.History {
		filename := filepath.Clean(h.History[i].Filename)
		if filename == oldFile {
			h.History[i].Filename = newFile
		} else if strings.HasPrefix(filename, oldFile+string(filepath.Separator)) {
			h.History[i].Filename = filepath.Join(newFile, strings.TrimPrefix(filename, oldFile+string(filepath.Separator)))
		}
	}

	setHistoryList(workspace, h)
	return nil
}

func HistoryListDelete(workspace, filename string) error {
	if filename == "" {
		return errors.New("no filename to delete from history")
	}

	filename = filepath.Clean(filename)

	h := getHistoryList(workspace)

	var err error
	for {
		i := -1
		for idx := range h.History {
			if filepath.Clean(h.History[idx].Filename) == filename {
				i = idx
				break
			}
		}
		if i == -1 {
			if len(h.History) == 0 {
				h.Index = 0
			} else if h.Index >= len(h.History) {
				h.Index = len(h.History) - 1
			} else if h.Index < 0 {
				h.Index = 0
			}
			setHistoryList(workspace, h)
			return nil
		}

		if i <= h.Index {
			h.Index--
		}

		h.History, err = lists.RemoveOrdered(h.History, i)
		if err != nil {
			return err
		}
	}
}

// Previous returns a filename or error
func HistoryListPrevious(workspace string) (string, error) {
	entry, err := HistoryListPreviousEntry(workspace)
	return entry.Filename, err
}

func HistoryListPreviousEntry(workspace string) (HistoryEntryT, error) {
	h := getHistoryList(workspace)

	if h.Index <= 0 || h.Index >= len(h.History) {
		return HistoryEntryT{}, errors.New("You already have the first history item open")
	}

	h.Index--
	setHistoryList(workspace, h)
	return h.History[h.Index], nil
}

// HistoryListNext returns a filename or error
func HistoryListNext(workspace string) (string, error) {
	entry, err := HistoryListNextEntry(workspace)
	return entry.Filename, err
}

func HistoryListNextEntry(workspace string) (HistoryEntryT, error) {
	h := getHistoryList(workspace)

	if h.Index < 0 || h.Index+1 >= len(h.History) {
		return HistoryEntryT{}, errors.New("You already have the last history item open")
	}

	h.Index++
	setHistoryList(workspace, h)
	return h.History[h.Index], nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
