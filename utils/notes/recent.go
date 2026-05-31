package notes

import (
	"errors"
	"fmt"
	"sort"

	"github.com/lmorg/murex/utils/lists"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/cache"
)

func getRecentList(projectRoot string) []string {
	var recent []string

	ok := cache.Read(cache.NS_NOTESW_RECENT, projectRoot, &recent)

	if !ok || len(recent) == 0 {
		return []string{}
	}

	return recent
}

func GetRecentList(projectRoot string) []string {
	recent := getRecentList(projectRoot)
	sort.Strings(recent)
	return recent
}

func setRecentList(projectRoot string, recent []string) {
	cache.Write(cache.NS_NOTESW_RECENT, projectRoot, &recent, cache.Days(365))
}

func RecentListAdd(projectRoot, filename string) error {
	if filename == "" {
		return errors.New("no filename to add to recent list")
	}

	recent := getRecentList(projectRoot)

	i := lists.MatchIndexString(recent, filename)
	if i > -1 {
		new, err := lists.RemoveOrdered(recent, i)
		if err != nil {
			return err
		}
		recent = new
	}

	recent = append(recent, filename)
	if len(recent) > config.Config.Notes.MaxRecentFiles {
		recent = recent[1:]
	}

	setRecentList(projectRoot, recent)
	return nil
}

func RecentListRename(projectRoot, oldFile, newFile string) error {
	if oldFile == "" || newFile == "" || oldFile == newFile {
		return fmt.Errorf("cannot update recent list: from '%s' to '%s'", oldFile, newFile)
	}

	recent := getRecentList(projectRoot)

	for i := range recent {
		if recent[i] != oldFile {
			recent[i] = newFile
			break
		}
	}

	setRecentList(projectRoot, recent)
	return nil
}

func RecentListDelete(projectRoot, filename string) error {
	if filename == "" {
		return errors.New("no filename to delete from recent list")
	}

	recent := getRecentList(projectRoot)

	i := lists.MatchIndexString(recent, filename)
	if i == -1 {
		return nil // nothing to delete
	}

	new, err := lists.RemoveOrdered(recent, i)
	if err != nil {
		return err
	}

	setRecentList(projectRoot, new)
	return nil
}
