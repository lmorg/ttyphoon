package webtools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
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

type duckDuckGoInputT struct {
	Query string `json:"query"`
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

type webScrapeInputT struct {
	URL string `json:"url"`
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
	return `Search the web using DuckDuckGo instant answer API. Input is a JSON object with a query string, for example {"query":"latest Go release"}.`
}
func (t *DuckDuckGoSearch) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

func (t *DuckDuckGoSearch) InputType() reflect.Type { return reflect.TypeOf(duckDuckGoInputT{}) }

func (t *DuckDuckGoSearch) Call(ctx context.Context, input string) (string, error) {
	query := strings.TrimSpace(input)
	return t.search(ctx, query)
}

func (t *DuckDuckGoSearch) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*duckDuckGoInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("searchDuckDuckGo received an invalid structured input %T", input)
	}
	return t.search(ctx, request.Query)
}

func (t *DuckDuckGoSearch) search(ctx context.Context, query string) (string, error) {
	query = strings.TrimSpace(query)
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

func (t *DuckDuckGoSearch) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*duckDuckGoInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", Inputs: []string{strings.TrimSpace(request.Query)}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	} else {
		observation.Summary = firstWebLine(output)
	}
	return observation
}

func firstWebLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
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
	return "Fetch and extract readable text content from a web page. Pass a JSON object with a url field directly as tool arguments, for example {\"url\":\"https://example.com\"}."
}
func (t *WebScrapePage) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}

func (t *WebScrapePage) InputType() reflect.Type { return reflect.TypeOf(webScrapeInputT{}) }

func (t *WebScrapePage) Call(ctx context.Context, input string) (string, error) {
	pageURL, err := parseURLInput(input)
	if err != nil {
		return "", err
	}
	return t.fetchPage(ctx, pageURL)
}

func (t *WebScrapePage) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*webScrapeInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("scrapeWebPage received an invalid structured input %T", input)
	}
	pageURL, err := validateWebURL(request.URL)
	if err != nil {
		return "", err
	}
	return t.fetchPage(ctx, pageURL)
}

func (t *WebScrapePage) fetchPage(ctx context.Context, pageURL string) (string, error) {
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

func (t *WebScrapePage) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*webScrapeInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", Inputs: []string{request.URL}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	} else {
		observation.Summary = firstWebLine(output)
	}
	return observation
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

	return validateWebURL(raw)
}

func validateWebURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("input URL cannot be empty")
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
