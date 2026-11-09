package hyperlink_tooltip

import (
	"fmt"
	"html"
	"strings"
)

type linkT struct {
	Scheme string
	Target string
	s      []string
}

func (l *linkT) Append(label, icon, value string) {
	l.s = append(l.s,
		fmt.Sprintf(`<div class="shared-tooltip-row"><span class="shared-tooltip-icon" role="img" aria-label="%s">&#x%s;</span> <code>%s</code></div>`,
			html.EscapeString(label),
			icon,
			html.EscapeString(value),
		),
	)
}

func (l *linkT) String() string {
	return strings.Join(l.s, "\n")
}

func newLink(scheme, target string) *linkT {
	return &linkT{Scheme: scheme, Target: target}
}
