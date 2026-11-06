package mainwindow

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	godebug "runtime/debug"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/file/meta"
	"github.com/lmorg/ttyphoon/utils/file/watcher"
	"github.com/lmorg/ttyphoon/utils/grep"
	"github.com/lmorg/ttyphoon/utils/lsp"
	"github.com/lmorg/ttyphoon/utils/notes"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
)

type GetFileReturnT struct {
	Contents string `json:"contents"`
	Binary   bool   `json:"binary"`
	Error    string `json:"error"`
}

type NotesGrepResultT = grep.Result

type NotesGrepReturnT = grep.ReturnValue

type NotesGrepOptionsT = grep.Options

func (a *WApp) GetFile(filename string) GetFileReturnT {
	filename = a.filePath(filename)

	stat, err := os.Stat(filename)
	if err != nil {
		//log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}
	if stat.Size() > config.Config.Notes.MaxFileSize*1024*1024 {
		return GetFileReturnT{Error: "File too large to open"}
	}

	f, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}

	defer f.Close()
	defer godebug.FreeOSMemory()

	b, err := io.ReadAll(f)
	if err != nil {
		//log.Println(err)
		return GetFileReturnT{Error: err.Error()}
	}

	a.setMarkdownBaseDir(filepath.Dir(filename))
	watcher.Watch(filename)

	if bytes.Contains(b[:min(1024, len(b))], []byte{0}) {
		return GetFileReturnT{Contents: base64.StdEncoding.EncodeToString(b), Binary: true}
	}

	return GetFileReturnT{Contents: string(b), Binary: false}
}

// SaveFile saves a file. If projectPath is empty, it uses the current $PROJECT.
// If projectPath is set, $PROJECT in filename is expanded against projectPath,
// which keeps autosave bound to the project where the file was opened.
func (a *WApp) SaveFile(filename, contents, projectPath string) error {
	var absPath string
	if projectPath == "" {
		absPath = a.filePath(filename)
	} else {
		absPath = a.filePathWithProject(filename, projectPath)
	}

	dir := filepath.Dir(absPath)
	_, statErr := os.Stat(dir)
	created := os.IsNotExist(statErr)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}

	watcher.NoteWrite(absPath, []byte(contents))

	_, statErr = os.Stat(absPath)
	created = os.IsNotExist(statErr)

	if err := os.WriteFile(absPath, []byte(contents), 0644); err != nil {
		return err
	}
	notes.NotifyFileChanges(absPath)

	if created {
		a.notifyLspWorkspaceFiles(func(t *lsp.Transport) error {
			return lsp.NotifyDidCreateFiles(t, []string{absPath})
		})
	}

	return nil
}

func (a *WApp) SaveBinaryFile(filename, base64Contents string) error {
	filename = a.filePath(filename)

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("create directory for binary file: %w", err)
	}

	b, err := base64.StdEncoding.DecodeString(base64Contents)
	if err != nil {
		return fmt.Errorf("decode base64 file contents: %w", err)
	}

	if err := os.WriteFile(filename, b, 0644); err != nil {
		return err
	}
	notes.NotifyFileChanges(filename)
	return nil
}

func (a *WApp) SaveImageDialog(defaultFilename string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Image",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Images",
				Pattern:     "*.png;*.jpg;*.jpeg;*.gif;*.webp;*.svg",
			},
		},
	})
	if err != nil {
		return "", err
	}

	return path, nil
}

func (a *WApp) SaveCodeDialog(defaultFilename string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:            "Save Code Block",
		DefaultDirectory: a.GetCurrentProject(),
		DefaultFilename:  defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Code files", Pattern: "*.txt;*.md;*.go;*.js;*.ts;*.py;*.sh;*.json;*.yaml;*.yml;*.html;*.css"},
		},
	})
	if err != nil {
		return "", err
	}

	return path, nil
}

func (a *WApp) GetFileMetaMarkdown(filename string) string {
	return meta.DocumentForPath(a.filePath(filename))
}

func (a *WApp) ComposeNoteLocationPath(location, name string) string {
	return filepath.Join(location, name)
}

func (a *WApp) ResolveNoteLocation(path string) string {
	return a.notesPathForHistory(path)
}

func (a *WApp) RenameFile(oldPath, newPath string) error {
	oldAbsPath := a.filePath(oldPath)
	newAbsPath := a.filePath(newPath)
	err := os.Rename(oldAbsPath, newAbsPath)
	if err != nil {
		return err
	}
	notes.NotifyFileChanges(oldAbsPath, newAbsPath)

	err = notes.HistoryListRename(a.notesHistoryWorkspace(), oldAbsPath, newAbsPath)
	if err != nil {
		return err
	}

	a.notifyLspWorkspaceFiles(func(t *lsp.Transport) error {
		return lsp.NotifyDidRenameFiles(t, [][2]string{{oldAbsPath, newAbsPath}})
	})

	return nil
}

func (a *WApp) DeleteFile(filename string) error {
	absPath := a.filePath(filename)
	err := os.Remove(absPath)
	if err != nil {
		return err
	}
	notes.NotifyFileChanges(absPath)

	err = notes.HistoryListDelete(a.notesHistoryWorkspace(), absPath)
	if err != nil {
		return err
	}

	a.notifyLspWorkspaceFiles(func(t *lsp.Transport) error {
		return lsp.NotifyDidDeleteFiles(t, []string{absPath})
	})

	return nil
}
