package lsp

type createFileWire struct {
	URI string `json:"uri"`
}

type renameFileWire struct {
	OldURI string `json:"oldUri"`
	NewURI string `json:"newUri"`
}

type deleteFileWire struct {
	URI string `json:"uri"`
}

// NotifyDidCreateFiles sends workspace/didCreateFiles for the provided paths.
func NotifyDidCreateFiles(t *Transport, absPaths []string) error {
	files := make([]createFileWire, 0, len(absPaths))
	for _, absPath := range absPaths {
		if absPath == "" {
			continue
		}
		files = append(files, createFileWire{URI: FilePathToURI(absPath)})
	}
	if len(files) == 0 {
		return nil
	}

	return t.Notify("workspace/didCreateFiles", map[string]any{"files": files})
}

// NotifyDidRenameFiles sends workspace/didRenameFiles for the provided path pairs.
func NotifyDidRenameFiles(t *Transport, pairs [][2]string) error {
	files := make([]renameFileWire, 0, len(pairs))
	for _, pair := range pairs {
		if pair[0] == "" || pair[1] == "" {
			continue
		}
		files = append(files, renameFileWire{
			OldURI: FilePathToURI(pair[0]),
			NewURI: FilePathToURI(pair[1]),
		})
	}
	if len(files) == 0 {
		return nil
	}

	return t.Notify("workspace/didRenameFiles", map[string]any{"files": files})
}

// NotifyDidDeleteFiles sends workspace/didDeleteFiles for the provided paths.
func NotifyDidDeleteFiles(t *Transport, absPaths []string) error {
	files := make([]deleteFileWire, 0, len(absPaths))
	for _, absPath := range absPaths {
		if absPath == "" {
			continue
		}
		files = append(files, deleteFileWire{URI: FilePathToURI(absPath)})
	}
	if len(files) == 0 {
		return nil
	}

	return t.Notify("workspace/didDeleteFiles", map[string]any{"files": files})
}
