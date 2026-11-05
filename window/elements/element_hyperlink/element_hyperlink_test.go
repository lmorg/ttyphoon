package element_hyperlink

import (
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

func TestGenerateHyperlinkURIs(t *testing.T) {
	for _, uri := range []string{
		"https://example.com/a;b?q=c;d", "file:///tmp/a.txt",
		"mailto:someone@example.com", "ssh:host", "custom+scheme:value", "file:///tmp/100% done.txt",
	} {
		element := &ElementHyperlink{}
		if err := element.Generate(types.NewApcSliceNoParse([]string{"label", uri})); err != nil {
			t.Errorf("%s: %v", uri, err)
		}
		if element.Url() != uri || element.Label() != "label" || element.Scheme() == "" {
			t.Errorf("URI or label not preserved: %#v", element)
		}
	}
	for _, uri := range []string{"", "relative/path", "1bad:value", "https://example.com/\n"} {
		if err := (&ElementHyperlink{}).Generate(types.NewApcSliceNoParse([]string{"label", uri})); err == nil {
			t.Errorf("invalid URI accepted: %q", uri)
		}
	}
}

func TestTooltipText(t *testing.T) {
	for _, test := range []struct {
		name   string
		scheme string
		path   string
		values []string
	}{
		{name: "https", scheme: "https", path: "example.com/docs/page.html", values: []string{"https", "example.com/docs/page.html"}},
		{name: "file", scheme: "file", path: "/project/docs/file.md", values: []string{"file", "/project/docs", "file.md"}},
		{name: "root file", scheme: "file", path: "/file.md", values: []string{"file", "/", "file.md"}},
		{name: "escaped file", scheme: "file", path: "/project/<docs>/a&b.md", values: []string{"file", "/project/&lt;docs&gt;", "a&amp;b.md"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			element := &ElementHyperlink{scheme: test.scheme, path: test.path}
			tooltip := element.tooltipText()
			if strings.Count(tooltip, `class="shared-tooltip-row"`) != len(test.values) {
				t.Fatalf("unexpected tooltip rows: %s", tooltip)
			}
			for _, value := range test.values {
				if !strings.Contains(tooltip, "<code>"+value+"</code>") {
					t.Errorf("tooltip missing value %q: %s", value, tooltip)
				}
			}
			for _, icon := range []string{"f5fc", "f07b"} {
				if !strings.Contains(tooltip, "&#x"+icon+";") {
					t.Errorf("tooltip missing icon %s: %s", icon, tooltip)
				}
			}
			if strings.Contains(tooltip, "&#xf15b;") != (test.scheme == "file") {
				t.Errorf("unexpected filename icon: %s", tooltip)
			}
		})
	}
}
