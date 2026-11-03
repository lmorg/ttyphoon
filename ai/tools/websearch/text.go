package websearch

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/lmorg/ttyphoon/app"
)

var defaultUserAgent = app.Name() + "/" + app.Version()

func firstWebLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

func searchResultText(raw string, limit int) string {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return ""
	}
	document.Find("script, style, noscript").Remove()
	text := []rune(strings.Join(strings.Fields(document.Text()), " "))
	if len(text) > limit {
		return string(text[:limit]) + "..."
	}
	return string(text)
}

func validateWebURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("input URL cannot be empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be http or https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("URL host cannot be empty")
	}

	return u.String(), nil
}
