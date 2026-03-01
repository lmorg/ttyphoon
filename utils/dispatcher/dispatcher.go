package dispatcher

const ENV_WINDOW = "MXTTY_WINDOW"

const ENV_PARAMETERS = "MXTTY_PARAMETERS"

type WindowTypeT string

const (
	WindowSdl      WindowTypeT = "sdl"
	WindowInputBox WindowTypeT = "inputBox"
	WindowMarkdown WindowTypeT = "markdown"
	WindowHistory  WindowTypeT = "history"
	WindowPreview  WindowTypeT = "preview"
)

type PInputBoxT struct {
	Title   string   `json:"title"`
	Prefill string   `json:"prefill"`
	History []string `json:"history"`
}

type PMarkdownT struct {
	Path string `json:"path"`
}

type PPreviewT struct{}
