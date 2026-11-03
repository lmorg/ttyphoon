package webtools

import (
	"context"
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

var defaultUserAgent = app.Name() + "/" + app.Version()

type WebScrapePage struct {
	agent   aitypes.Agent
	enabled bool
	client  *http.Client
}

type webScrapeInputT struct {
	URL string `json:"url"`
}

func init() {
	agent.ToolsAdd(&WebScrapePage{})
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

func firstWebLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
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
