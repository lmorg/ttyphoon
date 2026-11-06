package mainwindow

import (
	"strings"

	"github.com/lmorg/ttyphoon/utils/spellcheck"
)

// SpellCheckSuggestionT holds a misspelled word with correction suggestions.
// WordStart is the absolute character offset within the checked text.
type SpellCheckSuggestionT struct {
	MisspeltWord string   `json:"misspeltWord"`
	WordStart    int      `json:"wordStart"`
	WordLength   int      `json:"wordLength"`
	Suggestions  []string `json:"suggestions"`
}

// NotesSpellCheck runs the provided text through aspell and returns a list of
// misspelled words with suggestions. Returns nil if aspell is not available or
// an error occurs — callers should degrade gracefully.
func (a *WApp) NotesSpellCheck(text string) []SpellCheckSuggestionT {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	raw, err := spellcheck.ExecAspell(text)
	if err != nil {
		return nil
	}

	results, err := spellcheck.ParseAspellMultilineOutput(text, raw)
	if err != nil {
		return nil
	}

	out := make([]SpellCheckSuggestionT, 0, len(results))
	for _, s := range results {
		out = append(out, SpellCheckSuggestionT{
			MisspeltWord: s.MisspeltWord,
			WordStart:    s.WordStart,
			WordLength:   s.WordLength,
			Suggestions:  s.Suggestions,
		})
	}
	return out
}
