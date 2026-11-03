package websearch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"
)

func TestSearchResultsDeduplicateTrackingVariantsAndRetainMetadata(t *testing.T) {
	input := []searchResult{
		{Title: "First", URL: "https://EXAMPLE.org/story?id=1&utm_source=test", Engine: "alpha"},
		{Title: "Duplicate", URL: "https://example.org/story?fbclid=tracking&id=1", Engines: []string{"beta", "alpha"}, PublishedDate: "2026-10-16T09:00:00Z", Publisher: json.RawMessage(`{"name":"Publisher"}`), Source: json.RawMessage(`"Source feed"`)},
		{Title: "Distinct", URL: "https://example.org/story?id=2"},
		{Title: "Beyond limit", URL: "https://example.org/third"},
	}
	results := uniqueSearchResults(input, 2)
	if len(results) != 2 || results[0].URL != "https://example.org/story?id=1" || results[1].URL != "https://example.org/story?id=2" {
		t.Fatalf("deduplicated results = %+v", results)
	}
	output := formatSearchResult(results[0])
	for _, want := range []string{"First", "Published: 2026-10-16T09:00:00Z", "Publisher: Publisher", "Source: Source feed", "Engines: alpha, beta"} {
		if !strings.Contains(output, want) {
			t.Errorf("result %q is missing %q", output, want)
		}
	}
	if !strings.Contains(formatSearchResult(results[1]), "Published: unknown") {
		t.Fatal("missing publication date was not reported as unknown")
	}
}

func TestCanonicalSearchURLPreservesMeaningfulParametersAndFragments(t *testing.T) {
	got, err := canonicalSearchURL("https://example.org/item?lang=en&id=42&id=43&UTM_campaign=test&gclid=ad#/route")
	if err != nil || got != "https://example.org/item?id=42&id=43&lang=en#/route" {
		t.Fatalf("canonical URL = %q, %v", got, err)
	}
	for _, invalid := range []string{"javascript:alert(1)", "https://user:secret@example.org", "https://example.org?bad=%xx"} {
		if _, err := canonicalSearchURL(invalid); err == nil {
			t.Fatalf("invalid result URL accepted: %q", invalid)
		}
	}
}

func TestSearXNGPaginationMetadataAndRefinementsMakeOnlyOneRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path != "/search" || request.URL.Query().Get("pageno") != "2" || request.URL.Query().Get("q") != "news" {
			t.Errorf("pagination request = %s", request.URL)
		}
		_, _ = io.WriteString(writer, `{
"results":[
 {"title":"First","url":"https://example.org/news?id=1&utm_source=a","content":"News","engines":["alpha"],"publishedDate":"2026-10-16T10:00:00Z","publisher":"Publisher"},
 {"title":"Duplicate","url":"https://example.org/news?id=1&gclid=b","engine":"beta"},
 {"title":"Second","url":"https://example.org/news?id=2"},
 {"title":"Omitted","url":"https://example.org/over-limit"}
],
"answers":["<b>Legacy answer</b>",{"answer":"<b>Modern answer</b>","url":"https://answers.example.org/source?utm_medium=test"}],
"suggestions":["<b>better query</b>","better query"],
"corrections":["corrected query"],
"infoboxes":[
 {"infobox":"<b>Entity</b>","content":"Summary<script>injected()</script>","urls":[{"url":"javascript:alert(1)"},{"url":"https://reference.example.org/entity"}],"attributes":[{"label":"Founded","value":"2020"},{"label":"Count","value":42}],"img_src":"https://images.example.org/not-requested"},
 {"infobox":"Unattributed summary","content":"Must not present unsourced facts"}
]}`)
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "news", Page: 2, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Published: 2026-10-16T10:00:00Z", "Published: unknown", "Publisher: Publisher", "Engines: alpha, beta", "Page: 2; returned 2 unique results (limit 2)", "Legacy answer", "Modern answer", "Source: https://answers.example.org/source", "Infobox: Entity", "Founded: 2020", "Count: 42", "Source: https://reference.example.org/entity", "Suggested queries (not executed): better query", "Corrections (not executed): corrected query"} {
		if !strings.Contains(output, want) {
			t.Errorf("output %q is missing %q", output, want)
		}
	}
	for _, unwanted := range []string{"utm_", "gclid", "example.org/over-limit", "injected()", "<b>", "javascript:", "Unattributed summary", "images.example.org"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("output contains unwanted value %q: %s", unwanted, output)
		}
	}
	if strings.Count(output, "Suggested queries (not executed):") != 1 || requests.Load() != 1 {
		t.Fatalf("suggestions repeated or extra searches made: requests=%d output=%s", requests.Load(), output)
	}
}

func TestSearchPaginationAndLimitValidateBeforeNetwork(t *testing.T) {
	search := &DuckDuckGoSearch{provider: "searxng"}
	for _, input := range []*duckDuckGoInputT{
		{Query: "q", Page: -1}, {Query: "q", Page: 11},
		{Query: "q", Limit: -1}, {Query: "q", Limit: 51},
		{Action: "discover", Page: 1}, {Action: "discover", Limit: 1},
	} {
		if _, err := search.CallStructured(context.Background(), input); err == nil {
			t.Fatalf("invalid input accepted: %+v", input)
		}
	}
	if _, err := (&DuckDuckGoSearch{}).CallStructured(context.Background(), &duckDuckGoInputT{Query: "q", Page: 2}); err == nil {
		t.Fatal("DuckDuckGo accepted unsupported pagination")
	}
}

func TestSearchRefinementAndOutputBounds(t *testing.T) {
	suggestions := make([]string, 100)
	for index := range suggestions {
		suggestions[index] = strings.Repeat("x", index+1)
	}
	if output := formatSearchRefinements(suggestions, suggestions); strings.Count(output, "Suggested queries") != 5 || strings.Count(output, "Corrections") != 5 {
		t.Fatalf("refinement limits not applied: %q", output)
	}
	output := boundedSearchOutput(strings.Repeat("\u00e9", maxSearchOutputBytes))
	if len(output) > maxSearchOutputBytes || !utf8.ValidString(output) || !strings.Contains(output, "Search output truncated") {
		t.Fatalf("invalid bounded output: length=%d validUTF8=%t", len(output), utf8.ValidString(output))
	}
}
