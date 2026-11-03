package websearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const discoveryCacheTTL = 10 * time.Minute
const maxDiscoveryEngines = 256
const maxDiscoveryOutputBytes = 64 << 10

type discoveredEngine struct {
	Name             string   `json:"name"`
	Shortcut         string   `json:"shortcut"`
	Categories       []string `json:"categories"`
	Enabled          *bool    `json:"enabled,omitempty"`
	Paging           *bool    `json:"paging,omitempty"`
	LanguageSupport  *bool    `json:"language_support,omitempty"`
	TimeRangeSupport *bool    `json:"time_range_support,omitempty"`
	SafeSearch       *bool    `json:"safesearch,omitempty"`
}

type discoveryResult struct {
	InstanceName string             `json:"instance_name"`
	Version      string             `json:"version"`
	Categories   []string           `json:"categories"`
	Engines      []discoveredEngine `json:"engines"`
	EngineCount  int                `json:"engine_count"`
	Truncated    bool               `json:"truncated"`
}

func (t *DuckDuckGoSearch) discoverSearXNG(ctx context.Context, refresh bool) (string, error) {
	if refresh {
		t.discoveryMu.Lock()
		t.discoveryExpires = time.Time{}
		t.discoveryMu.Unlock()
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		t.discoveryMu.Lock()
		if t.discoveryData != "" && time.Now().Before(t.discoveryExpires) {
			data := t.discoveryData
			t.discoveryMu.Unlock()
			return data, nil
		}
		if pending := t.discoveryPending; pending != nil {
			t.discoveryMu.Unlock()
			select {
			case <-pending:
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		t.discoveryPending = make(chan struct{})
		t.discoveryMu.Unlock()
		data, err := t.fetchSearXNGDiscovery(ctx)
		t.discoveryMu.Lock()
		if err == nil {
			t.discoveryData = data
			t.discoveryExpires = time.Now().Add(discoveryCacheTTL)
		}
		close(t.discoveryPending)
		t.discoveryPending = nil
		t.discoveryMu.Unlock()
		return data, err
	}
}

func (t *DuckDuckGoSearch) fetchSearXNGDiscovery(ctx context.Context) (string, error) {
	endpoint, err := url.Parse(t.endpoint)
	if err != nil {
		return "", err
	}
	endpoint.Path = strings.TrimSuffix(endpoint.Path, "/search") + "/config"
	endpoint.RawPath = ""
	endpoint.RawQuery = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}
	response, err := t.doSearXNG(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("SearXNG discovery returned HTTP %d; the instance must expose /config. Normal search is still available", response.StatusCode)
	}
	var result discoveryResult
	if err := json.NewDecoder(io.LimitReader(response.Body, maxSearchResponseBytes)).Decode(&result); err != nil {
		return "", fmt.Errorf("cannot decode SearXNG /config JSON: %w", err)
	}
	if result.Engines == nil {
		return "", fmt.Errorf("SearXNG /config response is missing its engines list")
	}
	result.InstanceName = searchResultText(result.InstanceName, 150)
	result.Version = searchResultText(result.Version, 80)
	result.EngineCount = len(result.Engines)
	result.Truncated = false
	sort.Slice(result.Engines, func(left, right int) bool { return result.Engines[left].Name < result.Engines[right].Name })
	if len(result.Engines) > maxDiscoveryEngines {
		result.Engines = result.Engines[:maxDiscoveryEngines]
		result.Truncated = true
	}
	categoryCount := len(result.Categories)
	result.Categories = boundedDiscoveryNames(result.Categories, 64)
	if len(result.Categories) != categoryCount {
		result.Truncated = true
	}
	engines := result.Engines[:0]
	for _, engine := range result.Engines {
		if !validDiscoveryIdentifier(engine.Name, 120) {
			result.Truncated = true
			continue
		}
		if engine.Shortcut != "" && !validDiscoveryIdentifier(engine.Shortcut, 40) {
			engine.Shortcut = ""
			result.Truncated = true
		}
		categoryCount := len(engine.Categories)
		engine.Categories = boundedDiscoveryNames(engine.Categories, 32)
		if len(engine.Categories) != categoryCount {
			result.Truncated = true
		}
		engines = append(engines, engine)
	}
	result.Engines = engines
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	for len(encoded) > maxDiscoveryOutputBytes && len(result.Engines) > 0 {
		result.Truncated = true
		result.Engines = result.Engines[:len(result.Engines)/2]
		encoded, err = json.Marshal(result)
		if err != nil {
			return "", err
		}
	}
	return string(encoded), nil
}

func boundedDiscoveryNames(names []string, limit int) []string {
	bounded := make([]string, 0, min(len(names), limit))
	for _, name := range names {
		if validDiscoveryIdentifier(name, 120) {
			bounded = append(bounded, name)
		}
		if len(bounded) == limit {
			break
		}
	}
	sort.Strings(bounded)
	return bounded
}

func validDiscoveryIdentifier(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && len([]rune(value)) <= limit && !strings.ContainsAny(value, "\x00\r\n\t")
}
