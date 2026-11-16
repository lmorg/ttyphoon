package websearch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/envvars"
)

type scopedSearchTestAgent struct{}

type scopedSearchTestRenderer struct{ types.Renderer }

func (scopedSearchTestRenderer) DisplayNotification(types.NotificationType, string) {}

func (scopedSearchTestAgent) Renderer() types.Renderer { return scopedSearchTestRenderer{} }
func (scopedSearchTestAgent) ServiceName() string      { return "test" }
func (scopedSearchTestAgent) GetMeta() *aitypes.Meta   { return nil }
func (scopedSearchTestAgent) ProjectRoot() string      { return "" }
func (scopedSearchTestAgent) ServiceEnvironment() (*envvars.Environment, error) {
	return envvars.Resolve(nil, map[string]string{"TTYPHOON_WEB_SEARCH_PROVIDER": "duckduckgo"})
}
func (scopedSearchTestAgent) ImageGenerationEnvironment() (*envvars.Environment, error) {
	return envvars.Resolve(nil, nil)
}

func TestWebSearchUsesAIWideFilesAndRedactsHeaders(t *testing.T) {
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "duckduckgo")
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", `{"Authorization":"Bearer host-secret"}`)
	const secret = "scoped-search-secret"
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Header.Get("Authorization")
		_, _ = io.WriteString(writer, `{"results":[{"title":"`+secret+`","url":"https://example.org"}]}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.vars"), filepath.Join(dir, "second.vars")
	if err := os.WriteFile(first, []byte("TTYPHOON_WEB_SEARCH_PROVIDER=duckduckgo\n"), 0600); err != nil {
		t.Fatal(err)
	}
	content := "TTYPHOON_WEB_SEARCH_PROVIDER=searxng\nTTYPHOON_SEARXNG_URL=" + server.URL +
		"\nTTYPHOON_SEARXNG_HEADERS={\"Authorization\":\"Bearer " + secret + "\"}\nTTYPHOON_SEARXNG_MAX_CONCURRENT=3\n"
	if err := os.WriteFile(second, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	previous := config.Config.Ai.EnvFiles
	config.Config.Ai.EnvFiles = []string{first, second}
	t.Cleanup(func() { config.Config.Ai.EnvFiles = previous })
	tool, err := (&DuckDuckGoSearch{}).New(scopedSearchTestAgent{})
	if err != nil {
		t.Fatal(err)
	}
	search := tool.(*DuckDuckGoSearch)
	if search.provider != "searxng" || cap(search.gate.slots) != 3 {
		t.Fatal("web search did not use AI-wide file settings")
	}
	input := &duckDuckGoInputT{Query: "test"}
	output, err := search.CallStructured(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if <-requests != "Bearer "+secret || os.Getenv("TTYPHOON_WEB_SEARCH_PROVIDER") != "duckduckgo" {
		t.Fatal("web search did not use isolated scoped headers")
	}
	observation := search.ObservationStructured(input, output, nil)
	for _, text := range []string{output, observation.Summary, search.Description()} {
		if strings.Contains(text, secret) {
			t.Fatal("scoped search credential leaked")
		}
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Fatal("scoped search credential was not redacted")
	}
}

func TestWebSearchAIWideEmptyAndFileErrors(t *testing.T) {
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "searxng")
	t.Setenv("TTYPHOON_SEARXNG_URL", "http://127.0.0.1:13757")
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", `{"Authorization":"Bearer host-secret"}`)
	dir := t.TempDir()
	path := filepath.Join(dir, "search.vars")
	if err := os.WriteFile(path, []byte("TTYPHOON_WEB_SEARCH_PROVIDER=\n"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := config.Config.Ai.EnvFiles
	config.Config.Ai.EnvFiles = []string{path}
	t.Cleanup(func() { config.Config.Ai.EnvFiles = previous })
	tool, err := (&DuckDuckGoSearch{}).New(nil)
	if err != nil || tool.(*DuckDuckGoSearch).provider != "duckduckgo" {
		t.Fatalf("empty scoped provider did not override host: %v", err)
	}
	if err := os.WriteFile(path, []byte("TTYPHOON_SEARXNG_HEADERS=\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tool, err = (&DuckDuckGoSearch{}).New(nil)
	if err != nil || tool.(*DuckDuckGoSearch).headers.Get("Authorization") != "" {
		t.Fatalf("empty scoped headers did not override host: %v", err)
	}
	if err := os.WriteFile(path, []byte("TTYPHOON_SEARXNG_HEADERS=secret-invalid-header\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&DuckDuckGoSearch{}).New(nil); err == nil || strings.Contains(err.Error(), "secret-invalid-header") {
		t.Fatalf("invalid scoped headers error exposed its value: %v", err)
	}
	if err := os.WriteFile(path, []byte("TTYPHOON_WEB_SEARCH_PROVIDER=secret-invalid-provider\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&DuckDuckGoSearch{}).New(nil); err == nil || strings.Contains(err.Error(), "secret-invalid-provider") {
		t.Fatalf("invalid scoped provider error exposed its value: %v", err)
	}
	config.Config.Ai.EnvFiles = []string{filepath.Join(dir, "missing.vars")}
	if _, err := (&DuckDuckGoSearch{}).New(nil); err == nil || !strings.Contains(err.Error(), "resolve AI environment") {
		t.Fatalf("missing AI environment file was not reported: %v", err)
	}
}

func TestWebSearchRefreshesAIWideFilesAfterConfigReload(t *testing.T) {
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "searxng")
	t.Setenv("TTYPHOON_SEARXNG_URL", "http://127.0.0.1:1")
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", "")
	requests := make(chan string, 2)
	newServer := func() *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requests <- request.Header.Get("Authorization")
			_, _ = io.WriteString(writer, `{"results":[]}`)
		}))
	}
	first, second := newServer(), newServer()
	defer first.Close()
	defer second.Close()
	dir := t.TempDir()
	path1, path2 := filepath.Join(dir, "first.vars"), filepath.Join(dir, "second.vars")
	secondContent := "TTYPHOON_SEARXNG_URL=" + second.URL + "\nTTYPHOON_SEARXNG_HEADERS={\"Authorization\":\"Bearer second-key\"}\n"
	for path, content := range map[string]string{
		path1: "TTYPHOON_SEARXNG_URL=" + first.URL + "\nTTYPHOON_SEARXNG_HEADERS={\"Authorization\":\"Bearer first-key\"}\n",
		path2: secondContent,
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	previousConfig, previousDirs := config.Config, xdg.ConfigDirs
	config.Config.Ai = config.AiT{EnvFiles: []string{path1}}
	config.Config.Terminal.ColorTheme = ""
	config.Config.Notes.Languages = nil
	xdg.ConfigDirs = []string{dir}
	t.Cleanup(func() { config.Config, xdg.ConfigDirs = previousConfig, previousDirs })
	configDir := filepath.Join(dir, app.DirName)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "test.yaml")
	reload := func(path string) {
		t.Helper()
		if err := os.WriteFile(configPath, []byte(fmt.Sprintf("AI:\n  EnvFiles: [%q]\n", path)), 0600); err != nil {
			t.Fatal(err)
		}
		if err := config.LoadConfig(); err != nil {
			t.Fatal(err)
		}
	}
	tool, err := (&DuckDuckGoSearch{}).New(nil)
	if err != nil {
		t.Fatal(err)
	}
	search := tool.(*DuckDuckGoSearch)
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "first"}); err != nil {
		t.Fatal(err)
	}
	if <-requests != "Bearer first-key" {
		t.Fatal("first search did not use the initial AI file")
	}
	reload(path2)
	if err := os.Remove(path2); err != nil {
		t.Fatal(err)
	}
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "missing"}); err == nil {
		t.Fatal("search reused old credentials after a reloaded file went missing")
	}
	if err := os.WriteFile(path2, []byte(secondContent), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "second"}); err != nil {
		t.Fatal(err)
	}
	if <-requests != "Bearer second-key" {
		t.Fatal("existing search tool did not use the reloaded AI file")
	}
}

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
			if search.endpoint != test.endpoint || search.Name() != "webSearch" {
				t.Fatalf("search configuration = endpoint:%q name:%q", search.endpoint, search.Name())
			}
			if test.provider != "searxng" && !strings.Contains(search.Description(), "DuckDuckGo") {
				t.Fatal("description does not identify the active search provider")
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
