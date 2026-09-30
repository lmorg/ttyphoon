package webtools

import (
	"context"
	"reflect"
	"testing"
)

func TestWebToolsExposeStructuredInputTypes(t *testing.T) {
	search := &DuckDuckGoSearch{}
	searchType := search.InputType()
	if searchType.Kind() != reflect.Struct {
		t.Fatalf("searchDuckDuckGo InputType() = %v, want object", searchType)
	}
	if _, ok := searchType.FieldByName("Query"); !ok {
		t.Fatalf("searchDuckDuckGo input type has no query field: %v", searchType)
	}

	scrape := &WebScrapePage{}
	inputType := scrape.InputType()
	if inputType.Kind() != reflect.Struct {
		t.Fatalf("scrapeWebPage InputType() = %v, want object", inputType)
	}
	urlField, ok := inputType.FieldByName("URL")
	if !ok || urlField.Tag.Get("json") != "url" {
		t.Fatalf("scrapeWebPage input type has no JSON url field: %v", inputType)
	}
}

func TestWebToolsStructuredCallsValidateBeforeNetworkAccess(t *testing.T) {
	search := &DuckDuckGoSearch{}
	result, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "  "})
	if err != nil || result != "ERROR: query cannot be empty" {
		t.Fatalf("searchDuckDuckGo CallStructured() = %q, %v", result, err)
	}

	scrape := &WebScrapePage{}
	_, err = scrape.CallStructured(context.Background(), &webScrapeInputT{URL: "ftp://example.com"})
	if err == nil || err.Error() != "URL scheme must be http or https" {
		t.Fatalf("scrapeWebPage CallStructured() error = %v, want HTTP(S) scheme validation", err)
	}
}
