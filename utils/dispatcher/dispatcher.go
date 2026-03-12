package dispatcher

const ENV_WINDOW = "MXTTY_WINDOW"

const ENV_PARAMETERS = "MXTTY_PARAMETERS"

type WindowTypeT string

const (
	WindowTerminal WindowTypeT = "terminal"
	WindowInputBox WindowTypeT = "inputBox"
	WindowMarkdown WindowTypeT = "markdown"
	WindowHistory  WindowTypeT = "history"
	WindowPreview  WindowTypeT = "preview"
	WindowNotes    WindowTypeT = "notes"
)

type PInputBoxT struct {
	Title        string   `json:"title"`
	Prefill      string   `json:"prefill"`
	Placeholder  string   `json:"placeholder"`
	History      []string `json:"history"`
	NotesDisplay bool     `json:"notesDisplay"`
	NotesDefault bool     `json:"notesDefault"`
}

type PMarkdownT struct {
	Path string `json:"path"`
}

type PPreviewT struct{}

type PNotesT struct {
	ProjectRoot string `json:"projectRoot"`
	UserNotes   string `json:"userNotes"`
	Title       string `json:"title"`
	Filename    string `json:"filename"`
	Content     string `json:"content"`
}
