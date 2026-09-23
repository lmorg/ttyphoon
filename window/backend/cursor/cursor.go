package cursor

import "sync"

var (
	mu         sync.Mutex
	fn         func(string)
	base       = "default"
	current    string
	overridden bool
)

// Register sets the backend function that receives CSS cursor name changes.
// Call this once during renderer initialisation.
func Register(setCursor func(cursorCSS string)) {
	mu.Lock()
	defer mu.Unlock()

	fn = setCursor
	base = "default"
	current = ""
	overridden = false
}

func setLocked(css string) {
	if current == css || fn == nil {
		return
	}
	fn(css)
	current = css
}

func SetBase(css string) {
	mu.Lock()
	defer mu.Unlock()

	base = css
	if !overridden {
		setLocked(base)
	}
}

func ActivateBase(css string) {
	mu.Lock()
	defer mu.Unlock()

	base = css
	overridden = false
	setLocked(base)
}

func setOverride(css string) {
	mu.Lock()
	defer mu.Unlock()

	overridden = true
	setLocked(css)
}

func Arrow() {
	mu.Lock()
	defer mu.Unlock()

	overridden = false
	setLocked(base)
}

func Ibeam() { setOverride("text") }
func Hand()  { setOverride("pointer") }
func Zoom()  { setOverride("zoom-in") }
