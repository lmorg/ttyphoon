package hyperlink_tooltip

import (
	"net/url"
	"strings"
)

func mailto(link *linkT) {
	parsedURL, err := url.Parse(link.Scheme + ":" + link.Target)
	if err != nil {
		mailtoDefault(link, "")
		return
	}

	address, err := url.PathUnescape(parsedURL.Opaque)
	if err != nil {
		mailtoDefault(link, "")
		return
	}

	mailtoDefault(link, address)
	appendURLDetails(link, parsedURL)
}

func mailtoDefault(link *linkT, address string) {
	link.Append("Scheme", "f121", link.Scheme)

	split := strings.Split(address, "@")
	if len(split) == 2 {
		link.Append("User", "f007", split[0])
		link.Append("Host", "f015", split[1])
	} else {
		link.Append("Email", "f0e0", link.Target)
	}
}
