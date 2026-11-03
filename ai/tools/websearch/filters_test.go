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
	"time"
)

func seedDiscoveryTestCache(t *testing.T, search *DuckDuckGoSearch, payload string) {
	t.Helper()
	var result discoveryResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		t.Fatal(err)
	}
	search.discoveryMu.Lock()
	search.discoveryData = payload
	search.discoveryExpires = time.Now().Add(time.Minute)
	search.discoveryMu.Unlock()
}

func TestCachedDiscoveryRejectsUnknownNamesWithoutNetwork(t *testing.T) {
	search := &DuckDuckGoSearch{provider: "searxng"}
	seedDiscoveryTestCache(t, search, discoveryFixture)
	for _, input := range []*duckDuckGoInputT{
		{Query: "example", Engines: []string{"misspelled"}},
		{Query: "example", Categories: []string{"misspelled"}},
	} {
		if _, err := search.CallStructured(context.Background(), input); err == nil || !strings.Contains(err.Error(), "cached discovery") {
			t.Fatalf("unknown name validation error = %v", err)
		}
	}
}

func TestCachedDiscoveryWarningsAndShortcutResolutionDoNotFetchConfig(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path != "/search" || request.URL.Query().Get("engines") != "custom engine" {
			t.Errorf("search did not resolve the shortcut or fetched configuration: %s", request.URL)
		}
		_, _ = io.WriteString(writer, `{"results":[]}`)
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	seedDiscoveryTestCache(t, search, `{"categories":["general"],"engines":[{"name":"custom engine","shortcut":"c","categories":["general"],"enabled":false,"paging":false,"language_support":false,"time_range_support":false}]}`)
	input := &duckDuckGoInputT{Query: "example", Engines: []string{"!c"}, Page: 2, Language: "en", TimeRange: "month"}
	output, err := search.CallStructured(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"disabled by default", "does not advertise pagination", "does not advertise time-range", "does not advertise language"} {
		if !strings.Contains(output, want) {
			t.Errorf("missing capability warning %q in %q", want, output)
		}
	}
	if requests.Load() != 1 || input.Engines[0] != "!c" {
		t.Fatal("advisory validation made extra requests or mutated the caller's input")
	}
}

func TestDiscoveryValidationIsOptionalForMissingExpiredOrTruncatedCache(t *testing.T) {
	for _, state := range []string{"missing", "expired", "truncated"} {
		t.Run(state, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				if request.URL.Path != "/search" {
					t.Errorf("implicit discovery request: %s", request.URL)
				}
				_, _ = io.WriteString(writer, `{"results":[]}`)
			}))
			defer server.Close()
			search := discoveryTestSearch(t, server.URL)
			if state != "missing" {
				seedDiscoveryTestCache(t, search, `{"categories":[],"engines":[],"truncated":true}`)
			}
			if state == "expired" {
				search.discoveryExpires = time.Now().Add(-time.Second)
			}
			output, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example", Engines: []string{"unlisted"}, Categories: []string{"unlisted"}})
			if err != nil || requests.Load() != 1 {
				t.Fatalf("optional validation blocked search or made extra requests: %q, %v", output, err)
			}
			if state == "truncated" && !strings.Contains(output, "truncated") {
				t.Fatal("incomplete cached discovery did not produce an advisory")
			}
		})
	}
}

func TestUnknownCapabilitiesDoNotBecomeUnsupported(t *testing.T) {
	search := &DuckDuckGoSearch{}
	seedDiscoveryTestCache(t, search, `{"categories":["general"],"engines":[{"name":"unknown capabilities","categories":["general"]}]}`)
	_, warnings, err := search.cachedFilterAdvice(&duckDuckGoInputT{Page: 2, TimeRange: "month", Language: "en"})
	if err != nil || len(warnings) != 0 {
		t.Fatalf("unknown capability flags produced warnings: %v, %v", warnings, err)
	}
}
