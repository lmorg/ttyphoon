package browser

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func Open(app *application.App, rawURL string) error {
	_, err := OpenManaged(app, rawURL)
	return err
}

func OpenManaged(app *application.App, rawURL string) (func(), error) {
	parsedURL, err := parseWebURL(rawURL)
	if err != nil {
		return nil, err
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:       parsedURL.Hostname(),
		URL:         parsedURL.String(),
		Width:       1024,
		Height:      768,
		AlwaysOnTop: config.Config.Window.AlwaysOnTop,
	})
	window.Focus()
	return window.Close, nil
}

func parseWebURL(rawURL string) (*url.URL, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	if (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" {
		return nil, fmt.Errorf("only absolute HTTP(S) URLs can be opened in a new window: %q", rawURL)
	}
	return parsedURL, nil
}
