package mainwindow

import (
	"context"
	"encoding/base64"

	"golang.design/x/clipboard"
)

type ClipboardData struct {
	Text  string `json:"text"`
	Image string `json:"image"`
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
	items := formattedClipboardItems(html, rtf, plainText)
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
