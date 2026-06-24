package jupyter

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

func BookId(projectRoot, filePath string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, projectRoot)
	_, _ = io.WriteString(h, "::")
	_, _ = io.WriteString(h, filePath)
	return fmt.Sprintf("notes-code-%x", h.Sum(nil))
}

func errCreatingTempFile(err error) error { return fmt.Errorf("error creating temp file: %w", err) }

func writeRenderedSource(tempDir, fileName, code string, binding *LanguageBindingT) error {
	filePath := filepath.Join(tempDir, fileName)

	err := os.MkdirAll(tempDir, 0700)
	if err != nil {
		return errCreatingTempFile(err)
	}

	tempFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0700)
	if err != nil {
		return errCreatingTempFile(err)
	}
	defer tempFile.Close()

	buf := bytes.NewBuffer([]byte{})
	tmpl, err := template.New(filePath).Funcs(templateFuncs(code)).Parse(binding.Template)
	if err != nil {
		return errCreatingTempFile(err)
	}
	err = tmpl.Execute(buf, map[string]string{"Code": code})
	if err != nil {
		return errCreatingTempFile(err)
	}

	_, err = tempFile.Write(buf.Bytes())
	if err != nil {
		return errCreatingTempFile(err)
	}

	err = tempFile.Close()
	if err != nil {
		return errCreatingTempFile(err)
	}

	return nil
}

type FormatCodeReturnT struct {
	Code     string
	FilePath string
	Err      string
	// HasFormatter is true when a FormatCommand was configured for the
	// resolved language and therefore attempted. When false the caller should
	// fall back to LSP formatting.
	HasFormatter bool
}

func FormatCode(ctx context.Context, bookId, codeId, pwd, code, langRuntime string) *FormatCodeReturnT {
	log.Printf(`[debug] FormatCode: bookId="%s", codeId="%s", pwd="%s", langRuntime="%s"`, bookId, codeId, pwd, langRuntime)

	binding := resolveFormatBinding(langRuntime)
	if binding == nil {
		// Unknown language; the caller may fall back to LSP.
		return &FormatCodeReturnT{}
	}

	tempDir := tempDir(bookId)
	fileName := fileName(codeId, binding)
	filePath := filepath.Join(tempDir, fileName)

	err := writeRenderedSource(tempDir, fileName, code, binding)
	if err != nil {
		return &FormatCodeReturnT{Err: err.Error(), HasFormatter: len(binding.FormatCommand) > 0}
	}

	if len(binding.FormatCommand) == 0 {
		// No format command configured, but the rendered temp file is still
		// available (e.g. so the caller can open it for LSP). HasFormatter is
		// false so the caller falls back to LSP formatting.
		return &FormatCodeReturnT{FilePath: filePath}
	}

	exitNum := execute(ctx, codeId, pwd, expandVars(binding.FormatCommand, filePath), nil)
	if exitNum != 0 {
		return &FormatCodeReturnT{FilePath: filePath, HasFormatter: true}
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf(`[error] cannot format node: %v`, err.Error())
		return &FormatCodeReturnT{FilePath: filePath, HasFormatter: true}
	}

	return &FormatCodeReturnT{Code: string(b), FilePath: filePath, HasFormatter: true}
}

// FormatNotesContent formats a complete document's content using the configured
// FormatCommand for the given language. Unlike FormatCode it writes the raw
// content to the temp file (no language template wrapping), so it is suitable
// for whole files such as the main Notes editor. HasFormatter is false when no
// FormatCommand is configured, signalling the caller to fall back to LSP.
func FormatNotesContent(ctx context.Context, bookId, codeId, pwd, code, language string) *FormatCodeReturnT {
	log.Printf(`[debug] FormatNotesContent: bookId="%s", codeId="%s", pwd="%s", language="%s"`, bookId, codeId, pwd, language)

	binding := resolveFormatBinding(language)
	if binding == nil || len(binding.FormatCommand) == 0 {
		return &FormatCodeReturnT{}
	}

	tempDir := tempDir(bookId)
	fileName := fileName(codeId, binding)
	filePath := filepath.Join(tempDir, fileName)

	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return &FormatCodeReturnT{Err: errCreatingTempFile(err).Error(), HasFormatter: true}
	}
	if err := os.WriteFile(filePath, []byte(code), 0700); err != nil {
		return &FormatCodeReturnT{Err: errCreatingTempFile(err).Error(), HasFormatter: true}
	}

	exitNum := execute(ctx, codeId, pwd, expandVars(binding.FormatCommand, filePath), nil)
	if exitNum != 0 {
		return &FormatCodeReturnT{FilePath: filePath, HasFormatter: true}
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf(`[error] cannot format file: %v`, err.Error())
		return &FormatCodeReturnT{FilePath: filePath, HasFormatter: true}
	}

	return &FormatCodeReturnT{Code: string(b), FilePath: filePath, HasFormatter: true}
}

func FormatFile(ctx context.Context, pwd, filePath, lang string) {
	binding := getBindingByAlias(lang)
	if binding == nil || len(binding.FormatCommand) == 0 {
		log.Printf(`[debug] FormatFile: no bindings for %s`, lang)
		return
	}

	exitCode := execute(ctx, filePath, pwd, expandVars(binding.FormatCommand, filePath), nil)
	log.Printf(`[debug] FormatFile: exitCode=%d`, exitCode)
}
