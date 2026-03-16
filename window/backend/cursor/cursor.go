package cursor

var (
	fn      func(string)
	current string
)

// Register sets the backend function that receives CSS cursor name changes.
// Call this once during renderer initialisation.
func Register(setCursor func(cursorCSS string)) {
	fn = setCursor
	current = ""
}

func set(css string) {
	if current == css || fn == nil {
		return
	}
	fn(css)
	current = css
}

func Arrow() { set("default") }
func Ibeam() { set("text") }
func Hand()  { set("pointer") }
