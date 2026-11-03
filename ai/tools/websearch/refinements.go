package websearch

import (
	"encoding/json"
	"strings"
)

const maxSearchOutputBytes = 64 << 10

type searchSource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type searchInfobox struct {
	Title      string         `json:"infobox"`
	Content    string         `json:"content"`
	ID         string         `json:"id"`
	URLs       []searchSource `json:"urls"`
	Attributes []struct {
		Label string          `json:"label"`
		Value json.RawMessage `json:"value"`
	} `json:"attributes"`
}

func formatSearchAnswer(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if text = searchResultText(text, 1500); text != "" {
			return "Answer (source not supplied): " + text
		}
		return ""
	}
	var answer struct {
		Answer string `json:"answer"`
		URL    string `json:"url"`
	}
	if json.Unmarshal(raw, &answer) != nil {
		return ""
	}
	text = searchResultText(answer.Answer, 1500)
	if text == "" {
		return ""
	}
	if source, err := canonicalSearchURL(answer.URL); err == nil {
		return "Answer: " + text + "\n  Source: " + source
	}
	return "Answer (source not supplied): " + text
}

func formatSearchInfobox(box searchInfobox) string {
	sources := make([]string, 0, 3)
	seen := make(map[string]bool)
	for _, item := range append(box.URLs, searchSource{URL: box.ID}) {
		if source, err := canonicalSearchURL(item.URL); err == nil && !seen[source] {
			seen[source] = true
			sources = append(sources, "  Source: "+source)
		}
		if len(sources) >= 3 {
			break
		}
	}
	if len(sources) == 0 {
		return ""
	}
	title := searchResultText(box.Title, 200)
	if title == "" {
		title = "Entity summary"
	}
	lines := []string{"Infobox: " + title, "  " + searchResultText(box.Content, 1500)}
	for index, attribute := range box.Attributes {
		if index >= 8 {
			break
		}
		label := searchResultText(attribute.Label, 120)
		var value string
		if json.Unmarshal(attribute.Value, &value) != nil {
			var number json.Number
			if json.Unmarshal(attribute.Value, &number) == nil {
				value = number.String()
			}
		}
		if value = searchResultText(value, 300); label != "" && value != "" {
			lines = append(lines, "  "+label+": "+value)
		}
	}
	return strings.Join(append(lines, sources...), "\n")
}

func formatSearchRefinements(suggestions, corrections []string) string {
	var lines []string
	for _, group := range []struct {
		label string
		items []string
	}{{"Suggested queries (not executed)", suggestions}, {"Corrections (not executed)", corrections}} {
		seen := make(map[string]bool)
		for _, item := range group.items {
			text := searchResultText(item, 200)
			if text != "" && !seen[text] {
				seen[text] = true
				lines = append(lines, group.label+": "+text)
			}
			if len(seen) >= 5 {
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func boundedSearchOutput(output string) string {
	if len(output) <= maxSearchOutputBytes {
		return output
	}
	const notice = "\nSearch output truncated; lower limit or narrow the query."
	return strings.ToValidUTF8(output[:maxSearchOutputBytes-len(notice)], "") + notice
}
