package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
)

const duckDuckGoURL = "https://api.duckduckgo.com/"

var defaultUserAgent = app.Name() + "/" + app.Version()

type duckDuckGoResponse struct {
	AbstractText  string                `json:"AbstractText"`
	AbstractURL   string                `json:"AbstractURL"`
	RelatedTopics []duckDuckGoTopicNode `json:"RelatedTopics"`
}

type duckDuckGoTopicNode struct {
	Text     string                `json:"Text"`
	FirstURL string                `json:"FirstURL"`
	Topics   []duckDuckGoTopicNode `json:"Topics"`
}

type DuckDuckGoSearch struct {
	agent      aitypes.Agent
	enabled    bool
	maxResults int
	client     *http.Client
}

type WebScrapePage struct {
	agent   aitypes.Agent
	enabled bool
	client  *http.Client
}

func init() {
	agent.ToolsAdd(&DuckDuckGoSearch{})
	agent.ToolsAdd(&WebScrapePage{})
}

func (t *DuckDuckGoSearch) New(agentInst aitypes.Agent) (aitypes.Tool, error) {
	return &DuckDuckGoSearch{
		agent:      agentInst,
		enabled:    true,
		maxResults: 10,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

func (t *DuckDuckGoSearch) Enabled() bool { return t.enabled }
func (t *DuckDuckGoSearch) Toggle()       { t.enabled = !t.enabled }
func (t *DuckDuckGoSearch) Name() string  { return "searchDuckDuckGo" }
func (t *DuckDuckGoSearch) Path() string  { return "internal" }

func (t *DuckDuckGoSearch) Description() string {
	return "Search the web using DuckDuckGo instant answer API. Input should be a plain-text search query."
}

func (t *DuckDuckGoSearch) Call(ctx context.Context, input string) (string, error) {
	query := strings.TrimSpace(input)
	if query == "" {
		return "ERROR: query cannot be empty", nil
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is running web search: %s", t.agent.ServiceName(), query))

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("no_html", "1")
	params.Set("skip_disambig", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, duckDuckGoURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Sprintf("ERROR: search returned HTTP %d", resp.StatusCode), nil
	}

	var payload duckDuckGoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	results := make([]string, 0, t.maxResults)
	if payload.AbstractText != "" {
		results = append(results, fmt.Sprintf("- %s (%s)", payload.AbstractText, payload.AbstractURL))
	}
	collectDuckDuckGoTopics(payload.RelatedTopics, &results, t.maxResults)

	if len(results) == 0 {
		return "No results found.", nil
	}

	return strings.Join(results, "\n"), nil
}

func collectDuckDuckGoTopics(nodes []duckDuckGoTopicNode, results *[]string, max int) {
	for _, node := range nodes {
		if len(*results) >= max {
			return
		}
		if node.Text != "" {
			*results = append(*results, fmt.Sprintf("- %s (%s)", node.Text, node.FirstURL))
		}
		if len(node.Topics) > 0 {
			collectDuckDuckGoTopics(node.Topics, results, max)
		}
	}
}

func (t *WebScrapePage) New(agentInst aitypes.Agent) (aitypes.Tool, error) {
	return &WebScrapePage{
		agent:   agentInst,
		enabled: true,
		client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (t *WebScrapePage) Enabled() bool { return t.enabled }
func (t *WebScrapePage) Toggle()       { t.enabled = !t.enabled }
func (t *WebScrapePage) Name() string  { return "scrapeWebPage" }
func (t *WebScrapePage) Path() string  { return "internal" }

func (t *WebScrapePage) Description() string {
	return "Fetch and extract readable text content from a web page. Input should be a URL string or JSON like {\"url\":\"https://example.com\"}."
}

func (t *WebScrapePage) Call(ctx context.Context, input string) (string, error) {
	pageURL, err := parseURLInput(input)
	if err != nil {
		return "", err
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO,
		fmt.Sprintf("%s is fetching web page: %s", t.agent.ServiceName(), pageURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Sprintf("ERROR: fetch returned HTTP %d", resp.StatusCode), nil
	}

	reader := io.LimitReader(resp.Body, 512*1024)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return "", err
	}

	doc.Find("script, style, noscript").Remove()
	text := strings.TrimSpace(doc.Text())
	if text == "" {
		return "No readable content found.", nil
	}

	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 12000 {
		text = text[:12000]
	}

	return text, nil
}

func parseURLInput(input string) (string, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return "", fmt.Errorf("input URL cannot be empty")
	}

	type payload struct {
		URL string `json:"url"`
	}

	if strings.HasPrefix(raw, "{") {
		var v payload
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return "", fmt.Errorf("invalid JSON input: %w", err)
		}
		raw = strings.TrimSpace(v.URL)
		if raw == "" {
			return "", fmt.Errorf("input JSON must contain non-empty url")
		}
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("URL scheme must be http or https")
	}
	if u.Host == "" {
		return "", fmt.Errorf("URL host cannot be empty")
	}

	return u.String(), nil
}
