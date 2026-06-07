package jupyter

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
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

func FormatCode(ctx context.Context, pwd, id, filePath, langRuntime string, ch chan *OutputT) {
	binding := getBindingByDescription(langRuntime)
	if binding == nil || len(binding.FormatCommand) == 0 {
		return
	}

	execute(ctx, id, pwd, expandVars(binding.FormatCommand, filePath), ch)
}
