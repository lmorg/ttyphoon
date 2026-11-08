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

func TestTooltipCachedDuringGeneration(t *testing.T) {
	element := &ElementHyperlink{}
	generate := func(uri string) error {
		return element.Generate(types.NewApcSliceNoParse([]string{"label", uri}))
	}
	if err := generate("https://example.com/first"); err != nil {
		t.Fatal(err)
	}
	cached := element.tooltip
	if !strings.Contains(cached, "<code>/first</code>") {
		t.Fatalf("tooltip not generated: %s", cached)
	}
	element.path = "example.com/changed"
	if err := element.Write('x'); err != nil {
		t.Fatal(err)
	}
	if element.tooltip != cached {
		t.Fatal("tooltip changed after generation")
	}
	if err := generate("https://example.com/second"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(element.tooltip, "<code>/second</code>") {
		t.Fatalf("regeneration did not replace tooltip: %s", element.tooltip)
	}
	if err := generate(""); err == nil {
		t.Fatal("expected invalid URL error")
	}
	if element.tooltip != "" {
		t.Fatal("invalid generation retained stale tooltip")
	}
}

func TestTooltipText(t *testing.T) {
	for _, test := range []struct {
		name   string
		scheme string
		path   string
		values []string
	}{
		{name: "https", scheme: "https", path: "example.com/docs/page.html", values: []string{"https", "example.com", "/docs/page.html"}},
		{name: "file", scheme: "file", path: "/project/docs/file.md", values: []string{"/project/docs", "file.md"}},
		{name: "root file", scheme: "file", path: "/file.md", values: []string{"/", "file.md"}},
		{name: "escaped file", scheme: "file", path: "/project/<docs>/a&b.md", values: []string{"/project/&lt;docs&gt;", "a&amp;b.md"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			element := &ElementHyperlink{}
			if err := element.Generate(types.NewApcSliceNoParse([]string{"label", test.scheme + "://" + test.path})); err != nil {
				t.Fatal(err)
			}
			tooltip := element.tooltip
			if strings.Count(tooltip, `class="shared-tooltip-row"`) != len(test.values) {
				t.Fatalf("unexpected tooltip rows: %s", tooltip)
			}
			for _, value := range test.values {
				if !strings.Contains(tooltip, "<code>"+value+"</code>") {
					t.Errorf("tooltip missing value %q: %s", value, tooltip)
				}
			}
			icons := []string{"f07b"}
			if test.scheme != "file" {
				icons = append(icons, "f121")
			}
			for _, icon := range icons {
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

func TestHTTPTooltipExplicitAuthorityProperties(t *testing.T) {
	for _, test := range []struct {
		uri      string
		ip       string
		username string
		password string
		hasPass  bool
	}{
		{uri: "https://example.com/path"},
		{uri: "http://192.0.2.1:8080/path", ip: "192.0.2.1"},
		{uri: "https://alice:secret@[2001:db8::1]:443/path", ip: "2001:db8::1", username: "alice", password: "secret", hasPass: true},
		{uri: "https://alice@example.com/path", username: "alice"},
		{uri: "https://alice:@example.com/path", username: "alice", hasPass: true},
		{uri: "https://:secret@example.com/path", password: "secret", hasPass: true},
		{uri: "https://a%3Cb:p%26q@example.com/path?q=value", username: "a&lt;b", password: "p&amp;q", hasPass: true},
		{uri: "https://example.com/path?user=alice&password=secret&ip=192.0.2.1"},
	} {
		t.Run(test.uri, func(t *testing.T) {
			element := &ElementHyperlink{}
			if err := element.Generate(types.NewApcSliceNoParse([]string{"label", test.uri})); err != nil {
				t.Fatal(err)
			}
			tooltip := element.tooltip
			for _, property := range []struct {
				label   string
				icon    string
				value   string
				present bool
			}{
				{"IP", "f1eb", test.ip, test.ip != ""},
				{"User", "f007", test.username, test.username != ""},
				{"Pass", "f084", test.password, test.hasPass},
			} {
				if got := strings.Contains(tooltip, "&#x"+property.icon+";"); got != property.present {
					t.Errorf("%s present = %v, want %v: %s", property.label, got, property.present, tooltip)
				}
				if property.present && !strings.Contains(tooltip, "<code>"+property.value+"</code>") {
					t.Errorf("missing %s value %q: %s", property.label, property.value, tooltip)
				}
			}
		})
	}
}
