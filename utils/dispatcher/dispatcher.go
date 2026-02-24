package dispatcher

const ENV_WINDOW = "MXTTY_WINDOW"

const ENV_PARAMETERS = "MXTTY_PARAMETERS"

type WindowTypeT string

const (
	WindowSdl      WindowTypeT = "sdl"
	WindowInputBox WindowTypeT = "inputBox"
	WindowMarkdown WindowTypeT = "markdown"
	WindowHistory  WindowTypeT = "history"
)

type PInputBoxT struct {
	Title string `json:"title"`
}
type RInputBoxT struct {
	Cancelled bool
	Value     string
}

type PMarkdownT struct {
	Path string `json:"path"`
}
type RMarkdownT struct {
	// no return value required
}
