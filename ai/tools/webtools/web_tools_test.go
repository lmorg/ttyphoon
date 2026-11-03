package webtools

import (
	"context"
	"reflect"
	"testing"
)

func TestWebToolsExposeStructuredInputTypes(t *testing.T) {
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
	scrape := &WebScrapePage{}
	_, err := scrape.CallStructured(context.Background(), &webScrapeInputT{URL: "ftp://example.com"})
	if err == nil || err.Error() != "URL scheme must be http or https" {
		t.Fatalf("scrapeWebPage CallStructured() error = %v, want HTTP(S) scheme validation", err)
	}
}
