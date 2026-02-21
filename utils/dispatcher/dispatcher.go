package dispatcher

const ENV_WINDOW = "MXTTY_WINDOW"

const ENV_PARAMETERS = "MXTTY_PARAMETERS"

type PInputBoxT struct {
	Title string `json:"title"`
}

type RInputBoxT struct {
	Cancelled bool
	Value     string
}
