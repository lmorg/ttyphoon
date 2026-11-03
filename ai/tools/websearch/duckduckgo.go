package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (t *DuckDuckGoSearch) searchDuckDuckGo(ctx context.Context, query string, limit int) (string, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, duckDuckGoURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", defaultUserAgent)
	response, err := t.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return fmt.Sprintf("ERROR: search returned HTTP %d", response.StatusCode), nil
	}
	var payload duckDuckGoResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxSearchResponseBytes)).Decode(&payload); err != nil {
		return "", err
	}
	candidates := make([]searchResult, 0, limit)
	if payload.AbstractText != "" {
		candidates = append(candidates, searchResult{Title: payload.AbstractText, URL: payload.AbstractURL})
	}
	collectDuckDuckGoTopics(payload.RelatedTopics, &candidates, 1024)
	results := make([]string, 0, limit)
	if payload.AbstractText != "" && strings.TrimSpace(payload.AbstractURL) == "" {
		results = append(results, "- "+searchResultText(payload.AbstractText, 2000)+" (source not supplied)")
	}
	for _, result := range uniqueSearchResults(candidates, limit-len(results)) {
		results = append(results, fmt.Sprintf("- %s (%s)", searchResultText(result.Title, 2000), result.URL))
	}
	if len(results) == 0 {
		return "No results found.", nil
	}
	return boundedSearchOutput(strings.Join(results, "\n")), nil
}

type duckDuckGoResponse struct {
	AbstractText  string                `json:"AbstractText"`
	AbstractURL   string                `json:"AbstractURL"`
	RelatedTopics []duckDuckGoTopicNode `json:"RelatedTopics"`
}

type duckDuckGoTopicNode struct {
	Text     string                `json:"Text"`
	FirstURL string                `json:"FirstURL"`
	Topics   []duckDuckGoTopicNode `json:"Topics"`
}

func collectDuckDuckGoTopics(nodes []duckDuckGoTopicNode, results *[]searchResult, max int) {
	for _, node := range nodes {
		if len(*results) >= max {
			return
		}
		if node.Text != "" {
			*results = append(*results, searchResult{Title: node.Text, URL: node.FirstURL})
		}
		if len(node.Topics) > 0 {
			collectDuckDuckGoTopics(node.Topics, results, max)
		}
	}
}
