package types

// TerminalPaneTab represents a non-tmux tab rendered in the terminal pane UI
// (for example, embedded Notes when the notes pane is collapsed).
type TerminalPaneTab struct {
	ID     string
	Name   string
	Active bool
}
