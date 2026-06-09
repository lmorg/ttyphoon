package syntaxcompletion

import "slices"

// ContextState describes the syntax context around a cursor position.
type ContextState struct {
	InString  bool
	InComment bool
}

// ContextClassifier detects cursor context for a language.
// Implementations should be fast and best-effort; they can return a zero state
// when the language is unsupported.
type ContextClassifier interface {
	Detect(language, source string, cursor int) (ContextState, error)
}

type contextLanguageProvider interface {
	SupportedLanguages() []string
}

// SupportedTreeSitterLanguages returns tree-sitter language ids that are
// compiled into the default backend for this build.
func SupportedTreeSitterLanguages() []string {
	langs := defaultTreeSitterLanguages()
	return slices.Clone(langs)
}

// SupportedTreeSitterLanguages returns tree-sitter language ids available to
// this engine instance's classifier.
func (e *Engine) SupportedTreeSitterLanguages() []string {
	if e == nil || e.classifier == nil {
		return nil
	}
	provider, ok := e.classifier.(contextLanguageProvider)
	if !ok {
		return nil
	}
	return slices.Clone(provider.SupportedLanguages())
}
