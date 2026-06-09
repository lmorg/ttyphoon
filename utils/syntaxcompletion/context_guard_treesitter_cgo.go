//go:build cgo

package syntaxcompletion

import (
	"fmt"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsbash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
	tscsharp "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	tsc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tscpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	tscss "github.com/tree-sitter/tree-sitter-css/bindings/go"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tshtml "github.com/tree-sitter/tree-sitter-html/bindings/go"
	tsjava "github.com/tree-sitter/tree-sitter-java/bindings/go"
	tsjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tsjson "github.com/tree-sitter/tree-sitter-json/bindings/go"
	tsjulia "github.com/tree-sitter/tree-sitter-julia/bindings/go"
	tsocaml "github.com/tree-sitter/tree-sitter-ocaml/bindings/go"
	tsphp "github.com/tree-sitter/tree-sitter-php/bindings/go"
	tspython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tsruby "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	tsrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tsscala "github.com/tree-sitter/tree-sitter-scala/bindings/go"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	tsverilog "github.com/tree-sitter/tree-sitter-verilog/bindings/go"
)

type treeSitterClassifier struct {
	languages map[string]*sitter.Language
}

func newDefaultContextClassifier() ContextClassifier {
	goLang := sitter.NewLanguage(tsgo.Language())
	jsonLang := sitter.NewLanguage(tsjson.Language())
	htmlLang := sitter.NewLanguage(tshtml.Language())
	bashLang := sitter.NewLanguage(tsbash.Language())
	cLang := sitter.NewLanguage(tsc.Language())
	cppLang := sitter.NewLanguage(tscpp.Language())
	csharpLang := sitter.NewLanguage(tscsharp.Language())
	cssLang := sitter.NewLanguage(tscss.Language())
	javaLang := sitter.NewLanguage(tsjava.Language())
	jsLang := sitter.NewLanguage(tsjavascript.Language())
	juliaLang := sitter.NewLanguage(tsjulia.Language())
	ocamlLang := sitter.NewLanguage(tsocaml.LanguageOCaml())
	phpLang := sitter.NewLanguage(tsphp.LanguagePHP())
	pythonLang := sitter.NewLanguage(tspython.Language())
	rubyLang := sitter.NewLanguage(tsruby.Language())
	rustLang := sitter.NewLanguage(tsrust.Language())
	scalaLang := sitter.NewLanguage(tsscala.Language())
	typescriptLang := sitter.NewLanguage(tstypescript.LanguageTypescript())
	tsxLang := sitter.NewLanguage(tstypescript.LanguageTSX())
	verilogLang := sitter.NewLanguage(tsverilog.Language())

	return &treeSitterClassifier{
		languages: map[string]*sitter.Language{
			"bash":       bashLang,
			"sh":         bashLang,
			"shell":      bashLang,
			"c":          cLang,
			"cpp":        cppLang,
			"c++":        cppLang,
			"csharp":     csharpLang,
			"c#":         csharpLang,
			"css":        cssLang,
			"go":         goLang,
			"html":       htmlLang,
			"java":       javaLang,
			"javascript": jsLang,
			"js":         jsLang,
			"json":       jsonLang,
			"julia":      juliaLang,
			"ocaml":      ocamlLang,
			"php":        phpLang,
			"python":     pythonLang,
			"py":         pythonLang,
			"ruby":       rubyLang,
			"rust":       rustLang,
			"scala":      scalaLang,
			"typescript": typescriptLang,
			"ts":         typescriptLang,
			"tsx":        tsxLang,
			"verilog":    verilogLang,
		},
	}
}

func defaultTreeSitterLanguages() []string {
	langs := []string{
		"bash",
		"c",
		"cpp",
		"csharp",
		"css",
		"go",
		"html",
		"java",
		"javascript",
		"json",
		"julia",
		"ocaml",
		"php",
		"python",
		"ruby",
		"rust",
		"scala",
		"typescript",
		"verilog",
	}
	sort.Strings(langs)
	return langs
}

func (c *treeSitterClassifier) SupportedLanguages() []string {
	return defaultTreeSitterLanguages()
}

func (c *treeSitterClassifier) Detect(language, source string, cursor int) (ContextState, error) {
	lang := c.languages[strings.ToLower(strings.TrimSpace(language))]
	if lang == nil {
		return ContextState{}, nil
	}
	if cursor < 0 || cursor > len(source) {
		return ContextState{}, fmt.Errorf("cursor out of range")
	}
	if len(source) == 0 {
		return ContextState{}, nil
	}

	parser := sitter.NewParser()
	defer parser.Close()

	if err := parser.SetLanguage(lang); err != nil {
		return ContextState{}, err
	}

	tree := parser.Parse([]byte(source), nil)
	if tree == nil {
		return ContextState{}, fmt.Errorf("tree-sitter parse failed")
	}
	defer tree.Close()

	offset := cursor
	if offset == len(source) {
		offset--
	}
	if offset < 0 {
		offset = 0
	}

	root := tree.RootNode()
	node := root.NamedDescendantForByteRange(uint(offset), uint(offset))
	if node == nil {
		node = root.DescendantForByteRange(uint(offset), uint(offset))
	}
	if node == nil {
		return ContextState{}, nil
	}

	state := ContextState{}
	for n := node; n != nil; n = n.Parent() {
		kind := strings.ToLower(n.Kind())
		if strings.Contains(kind, "comment") {
			state.InComment = true
		}
		if strings.Contains(kind, "string") {
			state.InString = true
		}
		if state.InComment && state.InString {
			break
		}
	}

	return state, nil
}
