//go:build !cgo

package syntaxcompletion

func newDefaultContextClassifier() ContextClassifier {
	return nil
}

func defaultTreeSitterLanguages() []string {
	return nil
}
