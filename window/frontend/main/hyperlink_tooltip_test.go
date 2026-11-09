package mainwindow

import (
	"testing"

	"github.com/lmorg/ttyphoon/utils/hyperlink_tooltip"
)

func TestGetHyperlinkTooltip(t *testing.T) {
	app := &WApp{}
	for _, test := range []struct{ url, scheme, target string }{
		{"HTTPS://192.0.2.1/a", "https", "192.0.2.1/a"},
		{"file:///tmp/100% done.txt", "file", "/tmp/100% done.txt"},
		{"mailto:person@example.com?subject=Hello#section", "mailto", "person@example.com?subject=Hello#section"},
	} {
		got, err := app.GetHyperlinkTooltip(test.url)
		if err != nil {
			t.Fatal(err)
		}
		if want := hyperlink_tooltip.HTML(test.scheme, test.target); got != want {
			t.Errorf("tooltip = %q, want %q", got, want)
		}
	}
	for _, invalid := range []string{"", "/relative", "1bad:url", "https://example.com/\n"} {
		if _, err := app.GetHyperlinkTooltip(invalid); err == nil {
			t.Errorf("accepted invalid URL %q", invalid)
		}
	}
}
