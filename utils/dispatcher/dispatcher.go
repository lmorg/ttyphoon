package dispatcher

const ENV_WINDOW = "MXTTY_WINDOW"

const ENV_PARAMETERS = "MXTTY_PARAMETERS"

type PInputBoxT struct {
	Title string `json:"title"`
}

type PMarkdownT struct {
	Uri string `json:"uri"`
}

type RInputBoxT struct {
	Cancelled bool
	Value     string
}
