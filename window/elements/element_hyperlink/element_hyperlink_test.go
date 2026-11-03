package element_hyperlink

import (
	"strings"
	"testing"
)

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
