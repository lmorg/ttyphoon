package notes

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/utils/cache"
)

type HistoryListT struct {
	History []string
	Index   int
}

func getHistoryList(projectRoot string) *HistoryListT {
	h := &HistoryListT{}

	cache.Read(cache.NS_NOTESW_HISTORY, projectRoot, h)

	return h
}

func setHistoryList(projectRoot string, h *HistoryListT) {
	cache.Write(cache.NS_NOTESW_HISTORY, projectRoot, h, cache.Days(30))
}

func HistoryListCurrent(projectRoot string) string {
	h := getHistoryList(projectRoot)
	if len(h.History) == 0 {
		return ""
	}
	if h.Index < 0 || h.Index >= len(h.History) {
		return ""
	}

	return h.History[h.Index]
}

func HistoryListAdd(projectRoot, filename string) error {
	if filename == "" {
		return errors.New("no filename to add to recent list")
	}

	filename = filepath.Clean(filename)

	h := getHistoryList(projectRoot)
	truncateTo := h.Index + 1
	if truncateTo < 0 {
		truncateTo = 0
	}
	if truncateTo > len(h.History) {
		truncateTo = len(h.History)
	}

	h.History = h.History[:truncateTo]

	h.History = append(h.History, filename)
	if len(h.History) > 50 {
		h.History = h.History[len(h.History)-50:]
	}

	h.Index = len(h.History) - 1
	setHistoryList(projectRoot, h)
	return nil
}

func HistoryListRename(projectRoot, oldFile, newFile string) error {
	if oldFile == "" || newFile == "" || oldFile == newFile {
		return fmt.Errorf("cannot update recent list: from '%s' to '%s'", oldFile, newFile)
	}

	oldFile = filepath.Clean(oldFile)
	newFile = filepath.Clean(newFile)

	h := getHistoryList(projectRoot)

	for i := range h.History {
		if filepath.Clean(h.History[i]) == oldFile {
			h.History[i] = newFile
		}
	}

	setHistoryList(projectRoot, h)
	return nil
}

func HistoryListDelete(projectRoot, filename string) error {
	if filename == "" {
		return errors.New("no filename to delete from recent list")
	}

	filename = filepath.Clean(filename)

	h := getHistoryList(projectRoot)

	var err error
	for {
		i := -1
		for idx := range h.History {
			if filepath.Clean(h.History[idx]) == filename {
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
			setHistoryList(projectRoot, h)
			return nil
		}

		if i < h.Index {
			h.Index--
		}

		h.History, err = lists.RemoveOrdered(h.History, i)
		if err != nil {
			return err
		}
	}
}

// Previous returns a filename or error
func HistoryListPrevious(projectRoot string) (string, error) {
	h := getHistoryList(projectRoot)

	if h.Index == 0 {
		return "", errors.New("You already have the first history item open")
	}

	h.Index--
	setHistoryList(projectRoot, h)
	return h.History[h.Index], nil
}

// HistoryListNext returns a filename or error
func HistoryListNext(projectRoot string) (string, error) {
	h := getHistoryList(projectRoot)

	if h.Index+1 == len(h.History) {
		return "", errors.New("You already have the last history item open")
	}

	h.Index++
	setHistoryList(projectRoot, h)
	return h.History[h.Index], nil
}
