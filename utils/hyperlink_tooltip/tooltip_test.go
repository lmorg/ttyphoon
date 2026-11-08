package hyperlink_tooltip

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestHTML(t *testing.T) {
	for _, test := range []struct {
		name   string
		scheme string
		target string
		want   []string
	}{
		{
			name: "unescaped file", scheme: "file", target: "/tmp/100% <done>.txt",
			want: []string{"<code>/tmp</code>", "<code>100% &lt;done&gt;.txt</code>"},
		},
		{
			name: "query escaping", scheme: "https", target: "example.com/page?%22%3E%3Ctag%3E=%3Cvalue%3E",
			want: []string{`aria-label="&#34;&gt;&lt;tag&gt;"`, "<code>&#34;&gt;&lt;tag&gt;: &lt;value&gt;</code>"},
		},
		{
			name: "malformed web target", scheme: "http", target: "example.com/100% <done>",
			want: []string{"<code>http</code>", "<code>example.com/100% &lt;done&gt;</code>"},
		},
		{
			name: "other scheme", scheme: "mailto", target: "person@example.com",
			want: []string{"<code>mailto</code>", "<code>person@example.com</code>"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := HTML(test.scheme, test.target)
			for _, want := range test.want {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q in %s", want, got)
				}
			}
		})
	}
}

func TestHTTPHostPortAndDefaultPath(t *testing.T) {
	for _, test := range []struct {
		target   string
		path     string
		hostname string
		ip       string
		port     string
	}{
		{target: "example.com", path: "/", hostname: "example.com"},
		{target: "example.com:8443?x=y", path: "/", hostname: "example.com", port: "8443"},
		{target: "192.0.2.1:80/docs", path: "/docs", ip: "192.0.2.1", port: "80"},
		{target: "[2001:db8::1]:443", path: "/", ip: "2001:db8::1", port: "443"},
		{target: "[fe80::1%25en0]", path: "/", ip: "fe80::1%en0"},
		{target: "/docs", path: "/docs"},
	} {
		t.Run(test.target, func(t *testing.T) {
			got := HTML("https", test.target)
			if !strings.Contains(got, "<code>"+test.path+"</code>") {
				t.Errorf("missing path %q: %s", test.path, got)
			}

			for _, property := range []struct{ icon, value string }{
				{"f015", test.hostname}, {"f1eb", test.ip}, {"f13d", test.port},
			} {
				if present := strings.Contains(got, "&#x"+property.icon+";"); present != (property.value != "") {
					t.Errorf("icon %s present = %v: %s", property.icon, present, got)
				}
				if property.value != "" && !strings.Contains(got, "<code>"+property.value+"</code>") {
					t.Errorf("missing value %q: %s", property.value, got)
				}
			}
		})
	}
}

func TestTooltipRowOrder(t *testing.T) {
	rowLabel := regexp.MustCompile(`aria-label="([^"]*)"`)
	for _, test := range []struct {
		name   string
		scheme string
		target string
		labels []string
	}{
		{
			name: "hostname with all optional fields", scheme: "https",
			target: "alice:secret@example.com:8443/docs?q=first&q=second",
			labels: []string{"Scheme", "Hostname", "Port", "Path", "User", "Pass", "q", "q"},
		},
		{
			name: "IP with all optional fields", scheme: "http",
			target: "alice:secret@[2001:db8::1]:8080/docs?q=value",
			labels: []string{"Scheme", "IP", "Port", "Path", "User", "Pass", "q"},
		},
		{
			name: "no optional fields", scheme: "https", target: "/docs",
			labels: []string{"Scheme", "Path"},
		},
		{
			name: "hostname without path", scheme: "https", target: "example.com",
			labels: []string{"Scheme", "Hostname", "Path"},
		},
		{
			name: "file", scheme: "file", target: "/tmp/file.txt",
			labels: []string{"Path", "File"},
		},
		{
			name: "malformed HTTP URL", scheme: "http", target: "example.com/100%",
			labels: []string{"Scheme", "Path"},
		},
		{
			name: "other scheme", scheme: "mailto", target: "person@example.com",
			labels: []string{"Scheme", "Path"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var labels []string
			for _, match := range rowLabel.FindAllStringSubmatch(HTML(test.scheme, test.target), -1) {
				labels = append(labels, match[1])
			}
			if !reflect.DeepEqual(labels, test.labels) {
				t.Errorf("row order = %v, want %v", labels, test.labels)
			}
		})
	}
}
