package grep

// Result represents a single search result with JSON tags for Wails binding.
type Result struct {
	FileName string   `json:"fileName"`
	Path     string   `json:"path"`
	Line     int      `json:"line"`
	Context  []string `json:"context"`
}

// ReturnValue represents the return value for grep searches with results and error.
type ReturnValue struct {
	Results []Result `json:"results"`
	Error   string   `json:"error"`
}

// Options configures the search behavior.
type Options struct {
	CaseSensitive bool
	Regex         bool
	WholeWord     bool
	FileFilter    string // Optional: filter results by file path/name using the find package syntax (plain words=AND, "or", "!", "rx" regexp, "g" glob)
}
