package websearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const discoveryFixture = `{"instance_name":"<b>Private search</b>","version":"1.0","categories":["science","general"],"engines":[{"name":"zeta","shortcut":"z","categories":["science"],"enabled":false},{"name":"alpha","shortcut":"a","categories":["science","general"],"enabled":true,"paging":true,"language_support":true,"time_range_support":true,"safesearch":false}],"plugins":[{"name":"not-needed"}]}`

func discoveryTestSearch(t *testing.T, endpoint string) *DuckDuckGoSearch {
	t.Helper()
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "searxng")
	t.Setenv("TTYPHOON_SEARXNG_URL", endpoint)
	tool, err := (&DuckDuckGoSearch{}).New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return tool.(*DuckDuckGoSearch)
}

func TestSearXNGDiscoveryCachesCapabilitiesAndSupportsRefresh(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Method != http.MethodGet || request.URL.Path != "/prefix/config" || request.URL.RawQuery != "" {
			t.Errorf("discovery request = %s %s", request.Method, request.URL)
		}
		if request.Header.Get("Accept") != "application/json" || request.Header.Get("User-Agent") != defaultUserAgent {
			t.Errorf("discovery headers = %v", request.Header)
		}
		_, _ = io.WriteString(writer, discoveryFixture)
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL+"/prefix/search")
	input := &duckDuckGoInputT{Action: "discover"}
	output, err := search.CallStructured(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	var result discoveryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.InstanceName != "Private search" || result.Version != "1.0" || result.EngineCount != 2 || len(result.Engines) != 2 || result.Truncated {
		t.Fatalf("discovered instance = %+v", result)
	}
	first, second := result.Engines[0], result.Engines[1]
	if first.Name != "alpha" || first.Shortcut != "a" || second.Name != "zeta" || first.Categories[0] != "general" {
		t.Fatalf("engines not preserved/sorted: %+v", result.Engines)
	}
	if first.Enabled == nil || !*first.Enabled || second.Enabled == nil || *second.Enabled || first.TimeRangeSupport == nil || !*first.TimeRangeSupport || first.LanguageSupport == nil || !*first.LanguageSupport || first.SafeSearch == nil || *first.SafeSearch || second.LanguageSupport != nil {
		t.Fatal("discovery lost enabled, disabled, or unknown capability flags")
	}
	if strings.Contains(output, "plugins") || strings.Contains(output, "not-needed") {
		t.Fatal("unrelated instance configuration leaked into discovery")
	}
	cached, err := search.CallStructured(context.Background(), input)
	if err != nil || cached != output || requests.Load() != 1 {
		t.Fatalf("cache miss: requests=%d error=%v", requests.Load(), err)
	}
	_, err = search.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover", Refresh: true})
	if err != nil || requests.Load() != 2 {
		t.Fatalf("refresh failed: requests=%d error=%v", requests.Load(), err)
	}
	search.discoveryMu.Lock()
	search.discoveryExpires = time.Now().Add(-time.Second)
	search.discoveryMu.Unlock()
	_, err = search.CallStructured(context.Background(), input)
	if err != nil || requests.Load() != 3 {
		t.Fatalf("expiry failed: requests=%d error=%v", requests.Load(), err)
	}
	observation := search.ObservationStructured(input, output, nil)
	if observation.Status != "ok" || len(observation.Summary) > 100 || observation.Inputs[0] != "discover" {
		t.Fatalf("discovery observation = %+v", observation)
	}
	other := discoveryTestSearch(t, server.URL+"/prefix")
	if _, err := other.CallStructured(context.Background(), input); err != nil || requests.Load() != 4 {
		t.Fatalf("new tool reused another tool's cache: requests=%d error=%v", requests.Load(), err)
	}
}

func TestSearXNGDiscoveryPreservesFilterIdentifiers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, `{"engines":[{"name":"custom <engine> & source","shortcut":"q<&","categories":["custom <category>"]}]}`)
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover"})
	if err != nil {
		t.Fatal(err)
	}
	var result discoveryResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Engines) != 1 || result.Engines[0].Name != "custom <engine> & source" || result.Engines[0].Shortcut != "q<&" || result.Engines[0].Categories[0] != "custom <category>" {
		t.Fatalf("filter identifiers were changed: %+v", result)
	}
}

func TestSearXNGDiscoveryCoalescesConcurrentRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(writer, discoveryFixture)
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	var group sync.WaitGroup
	start := make(chan struct{})
	for range 24 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover"}); err != nil {
				t.Error(err)
			}
		}()
	}
	close(start)
	group.Wait()
	if requests.Load() != 1 {
		t.Fatalf("concurrent discovery made %d requests, want one", requests.Load())
	}
}

func TestSearXNGDiscoveryErrorsDoNotDisableSearchOrCacheFailures(t *testing.T) {
	var discoveryCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/config":
			if discoveryCalls.Add(1) == 1 {
				writer.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = io.WriteString(writer, discoveryFixture)
		case "/search":
			_, _ = io.WriteString(writer, `{"results":[]}`)
		default:
			t.Errorf("unexpected request: %s", request.URL)
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	input := &duckDuckGoInputT{Action: "discover"}
	if _, err := search.CallStructured(context.Background(), input); err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("restricted discovery error = %v", err)
	}
	if output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example"}); err != nil || output != "No results found." {
		t.Fatalf("normal search after discovery failure = %q, %v", output, err)
	}
	if _, err := search.CallStructured(context.Background(), input); err != nil || discoveryCalls.Load() != 2 {
		t.Fatalf("failed discovery was cached: requests=%d error=%v", discoveryCalls.Load(), err)
	}
}

func TestSearXNGDiscoveryResponseValidationAndBounds(t *testing.T) {
	var large discoveryResult
	large.Engines = make([]discoveredEngine, maxDiscoveryEngines+1)
	for index := range large.Engines {
		large.Engines[index].Name = fmt.Sprintf("engine-%03d", index)
		large.Engines[index].Categories = make([]string, 32)
		for category := range large.Engines[index].Categories {
			large.Engines[index].Categories[category] = strings.Repeat("a", 120)
		}
	}
	largeBody, err := json.Marshal(large)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"missing endpoint", "Not found", 404, true},
		{"redirect", "Redirect", 302, true},
		{"HTML", "<html>Not JSON</html>", 200, true},
		{"missing engines", `{}`, 200, true},
		{"empty engines", `{"engines":[]}`, 200, false},
		{"bounded", string(largeBody), 200, false},
		{"oversized", `{"instance_name":"` + strings.Repeat("x", maxSearchResponseBytes) + `","engines":[]}`, 200, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Location", "/do-not-follow")
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()
			search := discoveryTestSearch(t, server.URL)
			output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover"})
			if (err != nil) != test.wantError {
				t.Fatalf("discovery = %q, %v", output, err)
			}
			if err == nil && len(output) > maxDiscoveryOutputBytes {
				t.Fatalf("discovery output too large: %d", len(output))
			}
			if test.name == "bounded" {
				var result discoveryResult
				if err := json.Unmarshal([]byte(output), &result); err != nil || !result.Truncated || result.EngineCount != maxDiscoveryEngines+1 {
					t.Fatalf("truncation metadata = %+v, %v", result, err)
				}
			}
		})
	}
}

func TestDiscoveryValidationAndCancelledWaiter(t *testing.T) {
	search := &DuckDuckGoSearch{}
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover"}); err == nil {
		t.Fatal("DuckDuckGo accepted SearXNG discovery")
	}
	search.provider = "searxng"
	for _, input := range []*duckDuckGoInputT{
		{Action: "unknown"}, {Action: "discover", Query: "must not be sent"}, {Query: "query", Refresh: true},
	} {
		if _, err := search.CallStructured(context.Background(), input); err == nil {
			t.Fatalf("invalid discovery/search input accepted: %+v", input)
		}
	}
	search.discoveryPending = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := search.CallStructured(ctx, &duckDuckGoInputT{Action: "discover"})
		done <- err
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waiting discovery error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled discovery waiter remained blocked")
	}
}
