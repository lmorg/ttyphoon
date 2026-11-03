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

func TestSearXNGAuthenticationAndRedactionOnSearchAndDiscovery(t *testing.T) {
	const token = "synthetic-bearer-secret-8347"
	const key = "synthetic-key-9876"
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", `{"Authorization":"Bearer `+token+`","X-API-Key":"`+key+`"}`)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.Header.Get("Authorization") != "Bearer "+token || request.Header.Get("X-API-Key") != key {
			t.Error("configured authentication headers were not sent")
		}
		switch request.URL.Path {
		case "/search":
			_ = json.NewEncoder(writer).Encode(map[string]any{"results": []map[string]string{{"title": "Example", "url": "https://example.org", "content": "Echoed " + token + " and " + key}}})
		case "/config":
			_ = json.NewEncoder(writer).Encode(map[string]any{"instance_name": key, "engines": []any{}})
		default:
			t.Errorf("unexpected endpoint: %s", request.URL.Path)
		}
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	for _, input := range []*duckDuckGoInputT{{Query: "example"}, {Action: "discover"}} {
		output, err := search.CallStructured(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		observation := search.ObservationStructured(input, output, nil)
		for _, text := range []string{output, observation.Summary, search.Description()} {
			if strings.Contains(text, token) || strings.Contains(text, key) {
				t.Fatal("configured credentials appeared in tool-visible data")
			}
		}
		if !strings.Contains(output, "[REDACTED]") {
			t.Fatal("echoed credential was not redacted")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("authenticated requests = %d, want two", calls.Load())
	}
}

func TestSearXNGAuthenticationConfigurationDoesNotExposeValues(t *testing.T) {
	const secret = "synthetic-config-secret"
	for _, raw := range []string{
		`{"Authorization":"Bearer ` + secret + `", invalid}`,
		`{"Authorization":"` + secret + `\r\nInjected: value"}`,
		`{"Host":"` + secret + `"}`,
		`{"Bad Header":"` + secret + `"}`,
		`{"Authorization":17}`, `null`, `[]`,
		`{"Authorization":"a","authorization":"b"}`,
	} {
		if _, err := searxngHeaders(raw); err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("invalid configuration error is missing or exposes a secret: %v", err)
		}
	}
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "searxng")
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", `{"Authorization":"Bearer `+secret+`"}`)
	for _, endpoint := range []string{"https://user:" + secret + "@example.org", "http://example.org", "https://example.org?token=" + secret} {
		t.Setenv("TTYPHOON_SEARXNG_URL", endpoint)
		if _, err := (&DuckDuckGoSearch{}).New(nil); err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("invalid authenticated URL error is missing or exposes a secret: %v", err)
		}
	}
}

func TestSearXNGSecretsNeverReachDefaultProvider(t *testing.T) {
	t.Setenv("TTYPHOON_WEB_SEARCH_PROVIDER", "duckduckgo")
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", `{"Authorization":"Bearer searx-only-secret","X-API-Key":"private-instance-key"}`)
	t.Setenv("TTYPHOON_SEARXNG_MAX_CONCURRENT", "invalid-but-not-used-for-DuckDuckGo")
	tool, err := (&DuckDuckGoSearch{}).New(nil)
	if err != nil {
		t.Fatal(err)
	}
	search := tool.(*DuckDuckGoSearch)
	search.client.Transport = webToolsTestTransport(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "" || request.Header.Get("X-API-Key") != "" {
			t.Fatal("SearXNG authentication was sent to DuckDuckGo")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"RelatedTopics":[]}`))}, nil
	})
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example"}); err != nil {
		t.Fatal(err)
	}
}

func TestSearXNGRedactionCoversBasicAuthAndCookieValues(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://example.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.SetBasicAuth("private-username", "private-password")
	request.Header.Set("Cookie", "session=private-session; other=private-other")
	search := &DuckDuckGoSearch{headers: request.Header}
	text := "private-username private-password private-session private-other " + request.Header.Get("Authorization")
	redacted := search.redactSecrets(text)
	for _, secret := range []string{"private-username", "private-password", "private-session", "private-other", request.Header.Get("Authorization")} {
		if strings.Contains(redacted, secret) {
			t.Fatal("a Basic-auth or cookie credential was not redacted")
		}
	}
}

func TestSearXNGTransportErrorsAndRedirectsDoNotLeakCredentials(t *testing.T) {
	const secret = "synthetic-transport-secret"
	t.Setenv("TTYPHOON_SEARXNG_HEADERS", `{"Authorization":"Bearer `+secret+`"}`)
	var redirected atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { redirected.Add(1) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL, http.StatusFound)
	}))
	defer server.Close()
	search := discoveryTestSearch(t, server.URL)
	if _, err := search.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example"}); err == nil || redirected.Load() != 0 {
		t.Fatalf("search followed a redirect or ignored its status: %v", err)
	}
	search.client.Transport = webToolsTestTransport(func(*http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("synthetic failure echoed Bearer %s", secret)
	})
	input := &duckDuckGoInputT{Query: "private query must not enter transport errors"}
	_, err := search.CallStructured(context.Background(), input)
	if err == nil || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), input.Query) {
		t.Fatalf("transport error exposed request data: %v", err)
	}
	if len(search.gate.slots) != 0 || strings.Contains(search.ObservationStructured(input, "", err).Error, secret) {
		t.Fatal("transport failure leaked a slot or credential")
	}
}

func TestSearXNGCooldownIsSharedAndExpiresWithoutAutomaticRetry(t *testing.T) {
	for _, status := range []int{429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if calls.Add(1) == 1 {
					writer.Header().Set("Retry-After", "45")
					writer.WriteHeader(status)
					return
				}
				if request.URL.Path == "/config" {
					_, _ = io.WriteString(writer, discoveryFixture)
				} else {
					_, _ = io.WriteString(writer, `{"results":[]}`)
				}
			}))
			defer server.Close()
			first := discoveryTestSearch(t, server.URL)
			second := discoveryTestSearch(t, server.URL)
			if first.gate != second.gate {
				t.Fatal("agents did not share the instance request gate")
			}
			var clock atomic.Int64
			clock.Store(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC).UnixNano())
			first.gate.now = func() time.Time { return time.Unix(0, clock.Load()) }
			_, err := first.CallStructured(context.Background(), &duckDuckGoInputT{Query: "example"})
			if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("HTTP %d", status)) || !strings.Contains(err.Error(), "45 seconds") {
				t.Fatalf("missing Retry-After guidance: %v", err)
			}
			if _, err := second.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover"}); err == nil || calls.Load() != 1 {
				t.Fatal("discovery bypassed the shared cooldown or a retry was made")
			}
			clock.Add(int64(46 * time.Second))
			if _, err := second.CallStructured(context.Background(), &duckDuckGoInputT{Action: "discover"}); err != nil || calls.Load() != 2 {
				t.Fatalf("request did not resume after cooldown: %v", err)
			}
		})
	}
}

func TestSearXNGConcurrencyWaitCanBeCancelled(t *testing.T) {
	t.Setenv("TTYPHOON_SEARXNG_MAX_CONCURRENT", "1")
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(writer, `{"results":`)
		writer.(http.Flusher).Flush()
		started <- struct{}{}
		<-release
		_, _ = io.WriteString(writer, `[]}`)
	}))
	defer server.Close()
	defer unblock()
	first := discoveryTestSearch(t, server.URL)
	second := discoveryTestSearch(t, server.URL)
	finished := make(chan error, 1)
	go func() {
		_, err := first.CallStructured(context.Background(), &duckDuckGoInputT{Query: "first"})
		finished <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first request did not start")
	}
	ctx, cancel := context.WithCancel(context.Background())
	queued := make(chan error, 1)
	go func() { _, err := second.CallStructured(ctx, &duckDuckGoInputT{Query: "second"}); queued <- err }()
	cancel()
	select {
	case err := <-queued:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("queued cancellation = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued request did not cancel")
	}
	if calls.Load() != 1 || len(first.gate.slots) != 1 {
		t.Fatal("concurrency gate allowed another request or released a body too soon")
	}
	unblock()
	if err := <-finished; err != nil || len(first.gate.slots) != 0 {
		t.Fatalf("response completion failed to release the slot: %v", err)
	}
}

func TestRetryAfterParsesSecondsAndHTTPDates(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		value string
		wait  time.Duration
	}{
		{"60", time.Minute}, {"0", 0}, {"172800", 48 * time.Hour},
		{now.Add(48 * time.Hour).Format(http.TimeFormat), 48 * time.Hour},
		{now.Add(-time.Minute).Format(http.TimeFormat), 0},
		{"", 30 * time.Second}, {"invalid", 30 * time.Second}, {"-1", 30 * time.Second},
	} {
		if got := retryAfterDeadline(test.value, now).Sub(now); got != test.wait {
			t.Errorf("Retry-After %q = %s, want %s", test.value, got, test.wait)
		}
	}
}
