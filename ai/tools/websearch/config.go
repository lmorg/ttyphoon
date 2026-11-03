package websearch

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

const duckDuckGoURL = "https://api.duckduckgo.com/"

const maxSearchResponseBytes = 2 << 20

func (t *DuckDuckGoSearch) New(agentInst aitypes.Agent) (aitypes.Tool, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("TTYPHOON_WEB_SEARCH_PROVIDER")))
	endpoint := duckDuckGoURL
	var headers http.Header
	var gate *searxngRequestGate
	switch provider {
	case "", "duckduckgo":
		provider = "duckduckgo"
	case "searxng":
		raw := strings.TrimSpace(os.Getenv("TTYPHOON_SEARXNG_URL"))
		if raw == "" {
			return nil, fmt.Errorf("TTYPHOON_SEARXNG_URL is required for the searxng search provider")
		}
		validated, err := validateWebURL(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid TTYPHOON_SEARXNG_URL: use an absolute HTTP(S) instance URL")
		}
		instance, _ := url.Parse(validated)
		if instance.RawQuery != "" || instance.ForceQuery || instance.Fragment != "" || instance.User != nil {
			return nil, fmt.Errorf("TTYPHOON_SEARXNG_URL must not contain credentials, a query, or a fragment; use TTYPHOON_SEARXNG_HEADERS for authentication")
		}
		instance.Path = strings.TrimRight(instance.Path, "/")
		if !strings.HasSuffix(instance.Path, "/search") {
			instance.Path += "/search"
		}
		endpoint = instance.String()
		headers, err = searxngHeaders(os.Getenv("TTYPHOON_SEARXNG_HEADERS"))
		if err != nil {
			return nil, err
		}
		if len(headers) > 0 && instance.Scheme != "https" {
			host := instance.Hostname()
			address := net.ParseIP(host)
			if !strings.EqualFold(host, "localhost") && (address == nil || !address.IsLoopback()) {
				return nil, fmt.Errorf("authenticated SearXNG instances require HTTPS unless using a loopback address")
			}
		}
		concurrent := 2
		if configured := strings.TrimSpace(os.Getenv("TTYPHOON_SEARXNG_MAX_CONCURRENT")); configured != "" {
			concurrent, err = strconv.Atoi(configured)
			if err != nil || concurrent < 1 || concurrent > 16 {
				return nil, fmt.Errorf("TTYPHOON_SEARXNG_MAX_CONCURRENT must be an integer between 1 and 16")
			}
		}
		gate, err = instanceRequestGate(endpoint, concurrent)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported TTYPHOON_WEB_SEARCH_PROVIDER %q: use duckduckgo or searxng", provider)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	if provider == "searxng" {
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return &DuckDuckGoSearch{
		agent:      agentInst,
		enabled:    true,
		maxResults: 10,
		provider:   provider,
		endpoint:   endpoint,
		client:     client,
		headers:    headers,
		gate:       gate,
	}, nil
}

func searxngHeaders(raw string) (http.Header, error) {
	headers := make(http.Header)
	if strings.TrimSpace(raw) == "" {
		return headers, nil
	}
	var values map[string]string
	if len(raw) > 32<<10 || json.Unmarshal([]byte(raw), &values) != nil || values == nil {
		return nil, fmt.Errorf("TTYPHOON_SEARXNG_HEADERS must be a JSON object of header names to string values (maximum 32 KiB)")
	}
	for name, value := range values {
		if !validAuthHeaderName(name) || strings.ContainsAny(value, "\r\n\x00") {
			return nil, fmt.Errorf("TTYPHOON_SEARXNG_HEADERS contains an invalid or reserved header; values are not shown")
		}
		for _, character := range []byte(value) {
			if (character < 32 && character != '\t') || character == 127 {
				return nil, fmt.Errorf("TTYPHOON_SEARXNG_HEADERS contains an invalid header value; values are not shown")
			}
		}
		if _, exists := headers[http.CanonicalHeaderKey(name)]; exists {
			return nil, fmt.Errorf("TTYPHOON_SEARXNG_HEADERS contains duplicate header names")
		}
		headers.Set(name, value)
	}
	return headers, nil
}

func validAuthHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, character := range name {
		if !(character >= 'a' && character <= 'z') && !(character >= 'A' && character <= 'Z') && !(character >= '0' && character <= '9') && !strings.ContainsRune("!#$%&'*+-.^_`|~", character) {
			return false
		}
	}
	switch strings.ToLower(name) {
	case "host", "content-length", "transfer-encoding", "connection", "proxy-connection", "proxy-authorization", "upgrade", "trailer", "te", "accept", "user-agent":
		return false
	}
	return true
}
