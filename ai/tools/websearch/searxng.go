package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (t *DuckDuckGoSearch) searchSearXNG(ctx context.Context, query string, input *duckDuckGoInputT) (string, error) {
	engines, warnings, err := t.cachedFilterAdvice(input)
	if err != nil {
		return "", err
	}
	endpoint, err := url.Parse(t.endpoint)
	if err != nil {
		return "", err
	}
	params := endpoint.Query()
	params.Set("q", query)
	params.Set("format", "json")
	if input.Page > 0 {
		params.Set("pageno", strconv.Itoa(input.Page))
	}
	if len(input.Categories) > 0 {
		params.Set("categories", strings.Join(input.Categories, ","))
	}
	if len(engines) > 0 {
		params.Set("engines", strings.Join(engines, ","))
	}
	if input.Language != "" {
		params.Set("language", input.Language)
	}
	if input.TimeRange != "" {
		params.Set("time_range", input.TimeRange)
	}
	endpoint.RawQuery = params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	response, err := t.doSearXNG(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("SearXNG returned HTTP 403; enable json in search.formats in the instance's settings.yml, and check instance access restrictions")
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("SearXNG search returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		Results             []searchResult      `json:"results"`
		Answers             []json.RawMessage   `json:"answers"`
		Suggestions         []string            `json:"suggestions"`
		Corrections         []string            `json:"corrections"`
		Infoboxes           []searchInfobox     `json:"infoboxes"`
		UnresponsiveEngines [][]json.RawMessage `json:"unresponsive_engines"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxSearchResponseBytes)).Decode(&payload); err != nil {
		return "", fmt.Errorf("cannot decode SearXNG JSON response; ensure the instance supports format=json: %w", err)
	}
	limit := t.resultLimit(input)
	results := make([]string, 0, limit)
	unique := uniqueSearchResults(payload.Results, limit)
	for _, result := range unique {
		results = append(results, formatSearchResult(result))
	}
	for index, answer := range payload.Answers {
		if index >= 3 {
			break
		}
		if text := formatSearchAnswer(answer); text != "" {
			results = append(results, text)
		}
	}
	for index, box := range payload.Infoboxes {
		if index >= 3 {
			break
		}
		if text := formatSearchInfobox(box); text != "" {
			results = append(results, text)
		}
	}
	if len(results) == 0 {
		if len(payload.UnresponsiveEngines) > 0 {
			return formatSearchRefinements(payload.Suggestions, payload.Corrections), fmt.Errorf("SearXNG returned no results and reported unavailable search engines; check the instance's engine status or retry with different engines")
		}
		results = append(results, "No results found.")
	}
	if refinement := formatSearchRefinements(payload.Suggestions, payload.Corrections); refinement != "" {
		results = append(results, refinement)
	}
	if len(unique) > 0 {
		results = append(results, fmt.Sprintf("Page: %d; returned %d unique results (limit %d). Further pages depend on engine support and are not fetched automatically.", max(1, input.Page), len(unique), limit))
	}
	if len(payload.UnresponsiveEngines) > 0 {
		results = append(results, "Some SearXNG engines were unavailable; these results may be incomplete.")
	}
	for _, warning := range warnings {
		results = append(results, "Warning: "+warning)
	}
	return boundedSearchOutput(strings.Join(results, "\n")), nil
}
