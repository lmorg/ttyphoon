package hyperlink_tooltip

import (
	"net/url"
	"strings"
)

func mailto(link *linkT) {
	parsedURL, err := url.Parse(link.Scheme + ":" + link.Target)
	if err != nil {
		mailtoFallback(link)
		return
	}

	escapedAddress := parsedURL.Opaque
	if escapedAddress == "" {
		escapedAddress = parsedURL.EscapedPath()
	}
	address, err := url.PathUnescape(escapedAddress)
	if err != nil {
		mailtoFallback(link)
		return
	}

	if _, err := url.ParseQuery(parsedURL.RawQuery); err != nil {
		mailtoFallback(link)
		return
	}

	link.Append("Scheme", "f121", link.Scheme)

	if address != "" {
		split := strings.Split(address, "@")
		if len(split) == 2 {
			link.Append("User", "f007", split[0])
			link.Append("Host", "f015", split[1])
		} else {
			link.Append("Email", "f0e0", address)
		}
	}

	appendURLDetails(link, parsedURL)
}

func mailtoFallback(link *linkT) {
	link.Append("Scheme", "f121", link.Scheme)
	link.Append("Path", "f07b", link.Target)
}
