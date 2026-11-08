package hyperlink_tooltip

import (
	"fmt"
	"html"
	"net/netip"
	"net/url"
	"path"
	"strings"
)

// HTML renders a URL's scheme and target (the portion after scheme://).
// File targets may contain unescaped spaces or percent signs.
func HTML(scheme, target string) string {
	row := func(label, icon, value string) string {
		return fmt.Sprintf(`<div class="shared-tooltip-row"><span class="shared-tooltip-icon" role="img" aria-label="%s">&#x%s;</span> <code>%s</code></div>`, html.EscapeString(label), icon, html.EscapeString(value))
	}
	var rows []string
	switch scheme {
	case "file":
		rows = append(rows,
			row("Path", "f07b", path.Dir(target)),
			row("File", "f15b", path.Base(target)),
		)
	case "http", "https":
		parsedURL, err := url.Parse(scheme + "://" + target)
		if err != nil {
			rows = append(rows,
				row("Scheme", "f121", scheme),
				row("Path", "f07b", target),
			)
		} else {
			urlPath := parsedURL.Path
			if urlPath == "" {
				urlPath = "/"
			}

			rows = append(rows, row("Scheme", "f121", scheme))
			if hostname := parsedURL.Hostname(); hostname != "" {
				if _, err := netip.ParseAddr(hostname); err == nil {
					rows = append(rows, row("IP", "f1eb", hostname))
				} else {
					rows = append(rows, row("Hostname", "f015", hostname))
				}
			}
			if port := parsedURL.Port(); port != "" {
				rows = append(rows, row("Port", "f13d", port))
			}
			rows = append(rows, row("Path", "f07b", urlPath))
			if parsedURL.User != nil {
				if username := parsedURL.User.Username(); username != "" {
					rows = append(rows, row("User", "f007", username))
				}
				if password, present := parsedURL.User.Password(); present {
					rows = append(rows, row("Pass", "f084", password))
				}
			}
			for k, v := range parsedURL.Query() {
				for i := range v {
					rows = append(rows, row(k, "3f", fmt.Sprintf("%s: %s", k, v[i])))
				}
			}
		}
	default:
		rows = append(rows,
			row("Scheme", "f121", scheme),
			row("Path", "f07b", target),
		)
	}
	return strings.Join(rows, "\n")
}
