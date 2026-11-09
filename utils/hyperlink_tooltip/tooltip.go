package hyperlink_tooltip

// HTML renders a URL's scheme and target (the portion after scheme://).
// File targets may contain unescaped spaces or percent signs.
func HTML(scheme, target string) string {
	link := newLink(scheme, target)

	switch scheme {
	case "file":
		file(link)
	case "mailto":
		mailto(link)
	case "http", "https":
		http(link)
	default:
		link.Append("Scheme", "f121", scheme)
		link.Append("Path", "f07b", target)
	}

	return link.String()
}
