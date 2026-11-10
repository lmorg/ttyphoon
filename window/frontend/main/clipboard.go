package mainwindow

import (
	"context"
	"encoding/base64"
	"runtime"

	"github.com/lmorg/ttyphoon/utils/copy_format"
	"golang.design/x/clipboard"
)

type ClipboardData struct {
	Text  string `json:"text"`
	Image string `json:"image"`
}

func (a *WApp) CopyFormattedSelection(source string) error {
	items, err := renderedClipboardItems(source, false)
	if err != nil {
		return err
	}
	return writeClipboardItems(items)
}

func (a *WApp) CopyFormattedTable(source string) error {
	items, err := renderedClipboardItems(source, true)
	if err != nil {
		return err
	}
	return writeClipboardItems(items)
}

func renderedClipboardItems(source string, tabDelimited bool) ([]clipboard.Item, error) {
	formats, err := copy_format.Convert(source)
	if err != nil {
		return nil, err
	}
	if !tabDelimited {
		formats.PlainText = formats.Markdown
	}
	return selectionClipboardItems(formats), nil
}

func chromiumClipboardType(goos string) string {
	switch goos {
	case "darwin":
		return "org.chromium.web-custom-data"
	case "windows":
		return "Chromium Web Custom MIME Data Format"
	default:
		return "chromium/x-web-custom-data"
	}
}

func selectionClipboardItems(formats copy_format.Formats) []clipboard.Item {
	return append(formattedClipboardItems(formats.HTML, formats.RTF, formats.PlainText),
		clipboard.Item{
			Format: clipboard.Register(chromiumClipboardType(runtime.GOOS)),
			Bytes:  copy_format.ChromiumData(formats),
		},
	)
}

func (a *WApp) ClipboardSetHTML(html, plainText string) error {
	return a.ClipboardSetFormatted(html, "", plainText)
}

func formattedClipboardItems(html, rtf, plainText string) []clipboard.Item {
	items := []clipboard.Item{}
	if rtf != "" {
		items = append(items, clipboard.Item{Format: clipboard.Register("text/rtf"), Bytes: []byte(rtf)})
	}
	return append(items,
		clipboard.Item{Format: clipboard.Register("text/html"), Bytes: []byte(html)},
		clipboard.Item{Format: clipboard.FmtText, Bytes: []byte(plainText)},
	)
}

func (a *WApp) ClipboardSetFormatted(html, rtf, plainText string) error {
	return writeClipboardItems(formattedClipboardItems(html, rtf, plainText))
}

func writeClipboardItems(items []clipboard.Item) error {
	options := make([]clipboard.Option, len(items))
	for position, item := range items {
		options[position] = item
	}
	_, err := clipboard.WriteAll(context.Background(), options...)
	return err
}

// GetClipboardData returns clipboard data as either text or a base64-encoded PNG image.
func (a *WApp) GetClipboardData() ClipboardData {
	b, _ := clipboard.Read(context.Background(), clipboard.FmtImage)
	if len(b) != 0 {
		return ClipboardData{Image: base64.StdEncoding.EncodeToString(b)}
	}

	b, _ = clipboard.Read(context.Background(), clipboard.FmtText)
	return ClipboardData{Text: string(b)}
}
