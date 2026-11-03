package websearch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type searchResult struct {
	Title         string          `json:"title"`
	URL           string          `json:"url"`
	Content       string          `json:"content"`
	PublishedDate string          `json:"publishedDate"`
	PubDate       string          `json:"pubdate"`
	Publisher     json.RawMessage `json:"publisher"`
	Source        json.RawMessage `json:"source"`
	Engine        string          `json:"engine"`
	Engines       []string        `json:"engines"`
}

func canonicalSearchURL(raw string) (string, error) {
	if len(raw) > 4096 {
		return "", fmt.Errorf("result URL exceeds 4096 bytes")
	}
	validated, err := validateWebURL(raw)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(validated)
	if err != nil {
		return "", err
	}
	if parsed.User != nil {
		return "", fmt.Errorf("result URL contains credentials")
	}
	params, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return "", err
	}
	for key := range params {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" || lower == "msclkid" {
			params.Del(key)
		}
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.RawQuery = params.Encode()
	return parsed.String(), nil
}

func uniqueSearchResults(input []searchResult, limit int) []searchResult {
	results := make([]searchResult, 0, limit)
	seen := make(map[string]int)
	for _, result := range input {
		canonical, err := canonicalSearchURL(result.URL)
		if err != nil {
			continue
		}
		result.URL = canonical
		result.Engines = resultEngineNames(result.Engines, result.Engine)
		if index, ok := seen[canonical]; ok {
			current := &results[index]
			current.Engines = resultEngineNames(append(current.Engines, result.Engines...), "")
			if current.PublishedDate == "" && current.PubDate == "" {
				current.PublishedDate, current.PubDate = result.PublishedDate, result.PubDate
			}
			if resultMetadataText(current.Publisher) == "" {
				current.Publisher = result.Publisher
			}
			if resultMetadataText(current.Source) == "" {
				current.Source = result.Source
			}
			continue
		}
		if len(results) < limit {
			seen[canonical] = len(results)
			results = append(results, result)
		}
	}
	return results
}

func resultEngineNames(names []string, engine string) []string {
	unique := make(map[string]bool)
	for _, name := range append(names, engine) {
		if name = searchResultText(name, 120); name != "" && len(unique) < 16 {
			unique[name] = true
		}
	}
	result := make([]string, 0, len(unique))
	for name := range unique {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func resultMetadataText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return searchResultText(text, 200)
	}
	var object struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &object) == nil {
		return searchResultText(object.Name, 200)
	}
	return ""
}

func formatSearchResult(result searchResult) string {
	title := searchResultText(result.Title, 300)
	if title == "" {
		title = result.URL
	}
	date := searchResultText(result.PublishedDate, 100)
	if date == "" {
		date = searchResultText(result.PubDate, 100)
	}
	if date == "" {
		date = "unknown"
	}
	lines := []string{fmt.Sprintf("- %s\n  URL: %s\n  %s\n  Published: %s", title, result.URL, searchResultText(result.Content, 2000), date)}
	for _, metadata := range []struct{ label, value string }{
		{"Publisher", resultMetadataText(result.Publisher)},
		{"Source", resultMetadataText(result.Source)},
		{"Engines", strings.Join(result.Engines, ", ")},
	} {
		if metadata.value != "" {
			lines = append(lines, "  "+metadata.label+": "+metadata.value)
		}
	}
	return strings.Join(lines, "\n")
}
