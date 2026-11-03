package websearch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestWebSearchExposeStructuredInputType(t *testing.T) {
	search := &DuckDuckGoSearch{}
	searchType := search.InputType()
	if searchType.Kind() != reflect.Struct {
		t.Fatalf("input type = %v, want object", searchType)
	}
	if _, ok := searchType.FieldByName("Query"); !ok {
		t.Fatal("search input has no query field")
	}
}

func TestWebSearchValidatesBeforeNetworkAccess(t *testing.T) {
	search := &DuckDuckGoSearch{}
	result, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "  "})
	if err != nil || result != "ERROR: query cannot be empty" {
		t.Fatalf("search = %q, %v", result, err)
	}
}

func TestSearchProviderConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, provider, instance, endpoint string
		wantError                          bool
	}{
		{"default", "", "", duckDuckGoURL, false},
		{"explicit DuckDuckGo", "duckduckgo", "", duckDuckGoURL, false},
		{"SearXNG base", "searxng", "http://localhost:8080/", "http://localhost:8080/search", false},
		{"SearXNG prefix", "searxng", "https://example.org/searx/", "https://example.org/searx/search", false},
		{"SearXNG endpoint", "searxng", "https://example.org/search", "https://example.org/search", false},
		{"missing URL", "searxng", "", "", true},
		{"invalid scheme", "searxng", "file:///tmp/search", "", true},
		{"missing host", "searxng", "https:///search", "", true},
		{"query in base", "searxng", "https://example.org/?q=secret", "", true},
		{"unknown provider", "other", "", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", test.provider)
			t.Setenv("TTYPHOON_SEARXNG_URL", test.instance)
			tool, err := (&DuckDuckGoSearch{}).New(nil)
			if (err != nil) != test.wantError {
				t.Fatalf("New() error = %v, want error %t", err, test.wantError)
			}
			if test.wantError {
				return
			}
			search := tool.(*DuckDuckGoSearch)
			if search.endpoint != test.endpoint || search.Name() != "searchDuckDuckGo" {
				t.Fatalf("search configuration = endpoint:%q name:%q", search.endpoint, search.Name())
			}
			if test.provider == "searxng" && !strings.Contains(search.Description(), "SearXNG") {
				t.Fatal("description does not identify the active search provider")
			}
		})
	}
}

func TestSearXNGSearchUsesJSONFiltersAndBoundedPlainTextResults(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		query := request.URL.Query()
		if request.URL.Path != "/prefix/search" || request.Method != http.MethodGet || query.Get("q") != "C++ & Go" || query.Get("format") != "json" {
			t.Errorf("unexpected search request: %s %s", request.Method, request.URL)
		}
		if query.Get("categories") != "general,science" || query.Get("engines") != "google,arxiv" || query.Get("language") != "en-US" || query.Get("time_range") != "month" {
			t.Errorf("missing search filters: %v", query)
		}
		if request.Header.Get("Accept") != "application/json" || request.Header.Get("User-Agent") != defaultUserAgent {
			t.Errorf("unexpected search headers: %v", request.Header)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"results":[{"title":"unsafe","url":"javascript:alert(1)"},{"title":"<b>Go &amp; C++</b>","url":"https://example.org/first","content":"A <strong>snippet</strong><script>hidden()</script>"},{"title":"Second","url":"https://example.org/second","content":"More"},{"title":"Ignored","url":"https://example.org/third"}],"unresponsive_engines":[["bing","timeout"]]}`)
	}))
	defer server.Close()
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "searxng")
	t.Setenv("TTYPHOON_SEARXNG_URL", server.URL+"/prefix")
	tool, err := (&DuckDuckGoSearch{}).New(nil)
	if err != nil {
		t.Fatal(err)
	}
	search := tool.(*DuckDuckGoSearch)
	search.maxResults = 2
	output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{
		Query: "  C++ & Go  ", Categories: []string{"general", "science"}, Engines: []string{"google", "arxiv"}, Language: "en-US", TimeRange: "month",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Go & C++", "https://example.org/first", "A snippet", "https://example.org/second", "may be incomplete"} {
		if !strings.Contains(output, want) {
			t.Errorf("output %q does not contain %q", output, want)
		}
	}
	for _, unwanted := range []string{"<b>", "hidden()", "javascript:", "example.org/third"} {
		if strings.Contains(output, unwanted) {
			t.Errorf("output %q contains unwanted text %q", output, unwanted)
		}
	}
	if requests != 1 {
		t.Fatalf("search made %d requests, want one", requests)
	}
}

func TestSearXNGResponsesAndErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		body, want string
		wantError  bool
	}{
		{"empty", 200, `{"results":[]}`, "No results found.", false},
		{"answer", 200, `{"results":[],"answers":["<b>42</b>"]}`, "Answer (source not supplied): 42", false},
		{"JSON disabled", 403, "Forbidden", "search.formats", true},
		{"rate limited", 429, "Too many requests", "HTTP 429", true},
		{"server error", 500, "Failed", "HTTP 500", true},
		{"redirect not followed", 302, "Redirect", "HTTP 302", true},
		{"HTML response", 200, "<html>Not JSON</html>", "format=json", true},
		{"engines unavailable", 200, `{"results":[],"unresponsive_engines":[["google","timeout"]]}`, "unavailable search engines", true},
		{"response too large", 200, `{"results":[{"content":"` + strings.Repeat("x", maxSearchResponseBytes) + `"}]}`, "cannot decode", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if test.status == http.StatusFound {
					writer.Header().Set("Location", "/unexpected-redirect")
				}
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()
			t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "searxng")
			t.Setenv("TTYPHOON_SEARXNG_URL", server.URL)
			tool, err := (&DuckDuckGoSearch{}).New(nil)
			if err != nil {
				t.Fatal(err)
			}
			input := &duckDuckGoInputT{Query: "example"}
			output, callErr := tool.(*DuckDuckGoSearch).CallStructured(context.Background(), input)
			if (callErr != nil) != test.wantError {
				t.Fatalf("search = %q, %v", output, callErr)
			}
			if callErr != nil {
				output = callErr.Error()
				if tool.(*DuckDuckGoSearch).ObservationStructured(input, "", callErr).Status != "error" {
					t.Fatal("failed search was not recorded as an error")
				}
			}
			if !strings.Contains(output, test.want) {
				t.Fatalf("output %q does not contain %q", output, test.want)
			}
		})
	}
}

func TestSearchCancellationAndFilterValidation(t *testing.T) {
	search := &DuckDuckGoSearch{provider: "searxng", endpoint: "http://127.0.0.1:1/search", client: &http.Client{}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := search.CallStructured(ctx, &duckDuckGoInputT{Query: "example"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled search error = %v", err)
	}
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example", TimeRange: "week"}); err == nil {
		t.Fatal("unsupported time range was accepted")
	}
	if _, err := (&DuckDuckGoSearch{}).CallStructured(context.Background(), &duckDuckGoInputT{Query: "example", Language: "en"}); err == nil {
		t.Fatal("SearXNG-only filter was silently ignored by DuckDuckGo")
	}
}

func TestSearchResultTextIsBounded(t *testing.T) {
	if got := searchResultText("<b>abcdefghij</b>", 4); got != "abcd..." {
		t.Fatalf("bounded snippet = %q", got)
	}
}

type webToolsTestTransport func(*http.Request) (*http.Response, error)

func (transport webToolsTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestDefaultSearchStillUsesDuckDuckGo(t *testing.T) {
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "")
	t.Setenv("TTYPHOON_SEARXNG_URL", "")
	tool, err := (&DuckDuckGoSearch{}).New(nil)
	if err != nil {
		t.Fatal(err)
	}
	search := tool.(*DuckDuckGoSearch)
	search.client.Transport = webToolsTestTransport(func(request *http.Request) (*http.Response, error) {
		query := request.URL.Query()
		if request.URL.Host != "api.duckduckgo.com" || query.Get("q") != "example" || query.Get("format") != "json" || query.Get("no_html") != "1" || query.Get("skip_disambig") != "1" {
			t.Errorf("DuckDuckGo request changed: %s", request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"AbstractText":"An answer","AbstractURL":"https://example.org","RelatedTopics":[{"Text":"Related","FirstURL":"https://example.org/related"}]}`)),
		}, nil
	})
	output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example"})
	if err != nil || !strings.Contains(output, "An answer (https://example.org)") || !strings.Contains(output, "Related (https://example.org/related)") {
		t.Fatalf("DuckDuckGo search = %q, %v", output, err)
	}
}

func TestDuckDuckGoKeepsUnattributedAbstractWithinLimit(t *testing.T) {
	search := &DuckDuckGoSearch{client: &http.Client{Transport: webToolsTestTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"AbstractText":"Direct answer","RelatedTopics":[{"Text":"Over limit","FirstURL":"https://example.org/extra"}]}`))}, nil
	})}}
	output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example", Limit: 1})
	if err != nil || !strings.Contains(output, "Direct answer (source not supplied)") || strings.Contains(output, "Over limit") {
		t.Fatalf("unattributed DuckDuckGo answer = %q, %v", output, err)
	}
}

func TestDuckDuckGoLimitCountsUniqueURLs(t *testing.T) {
	search := &DuckDuckGoSearch{client: &http.Client{Transport: webToolsTestTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"AbstractText":"First","AbstractURL":"https://example.org/story?id=1&utm_source=a","RelatedTopics":[{"Text":"Duplicate","FirstURL":"https://example.org/story?id=1&gclid=b"},{"Topics":[{"Text":"Second","FirstURL":"https://example.org/story?id=2"},{"Text":"Omitted","FirstURL":"https://example.org/third"}]}]}`))}, nil
	})}}
	output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example", Limit: 2})
	if err != nil || !strings.Contains(output, "First (https://example.org/story?id=1)") || !strings.Contains(output, "Second (https://example.org/story?id=2)") || strings.Contains(output, "Duplicate") || strings.Contains(output, "Omitted") {
		t.Fatalf("unique DuckDuckGo results = %q, %v", output, err)
	}
}
