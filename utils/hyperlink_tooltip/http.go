package hyperlink_tooltip

import (
	"fmt"
	"net/netip"
	"net/url"
)

func http(link *linkT) {
	parsedURL, err := url.Parse(link.Scheme + "://" + link.Target)
	if err != nil {
		link.Append("Scheme", "f121", link.Scheme)
		link.Append("Path", "f07b", link.Target)
		return

	}

	link.Append("Scheme", "f121", link.Scheme)

	if hostname := parsedURL.Hostname(); hostname != "" {
		if _, err := netip.ParseAddr(hostname); err == nil {
			link.Append("IP", "f1eb", hostname)
		} else {
			link.Append("Hostname", "f015", hostname)
		}
	}

	if port := parsedURL.Port(); port != "" {
		link.Append("Port", "f13d", port)
	}

	urlPath := parsedURL.Path
	if urlPath == "" {
		urlPath = "/"
	}
	link.Append("Path", "f07b", urlPath)

	if parsedURL.User != nil {
		if username := parsedURL.User.Username(); username != "" {
			link.Append("User", "f007", username)
		}
		if password, present := parsedURL.User.Password(); present {
			link.Append("Pass", "f084", password)
		}
	}

	appendURLDetails(link, parsedURL)
}

func appendURLDetails(link *linkT, parsedURL *url.URL) {
	for k, values := range parsedURL.Query() {
		for _, value := range values {
			link.Append(k, "3f", fmt.Sprintf("%s: %s", k, value))
		}
	}
	if parsedURL.Fragment != "" {
		link.Append("Bookmark", "f02e", parsedURL.Fragment)
	}
}
