package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/tmc/langchaingo/callbacks"
)

type ChromeScraper struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	enabled          bool
}

func init() {
	agent.ToolsAdd(&ChromeScraper{})
}

func (t ChromeScraper) New(meta *agent.Meta) (agent.Tool, error) {
	return &ChromeScraper{meta: meta, enabled: true}, nil
}

func (t *ChromeScraper) Enabled() bool { return t.enabled }
func (t *ChromeScraper) Toggle()       { t.enabled = !t.enabled }

func (t *ChromeScraper) Description() string {
	return `Loads a web page and returns the contents of its page.
Useful for checking online content.
The input for this tool is a URL to the web page.`
}

func (t *ChromeScraper) Name() string { return "Web Scraper" }
func (t *ChromeScraper) Path() string { return "internal" }

func (t *ChromeScraper) Call(ctx context.Context, input string) (response string, err error) {
	if debug.Trace {
		log.Printf("Agent tool '%s' input:\n%s", t.Name(), input)
		defer func() {
			log.Printf("Agent tool '%s' response:\n%s", t.Name(), response)
			log.Printf("Agent tool '%s' error: %v", t.Name(), err)
		}()
	}

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	chromeCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(input),
		chromedp.OuterHTML("body", &response, chromedp.ByQuery),
	)

	if err != nil {
		if debug.Trace {
			log.Printf("Agent tool Chrome '%s' error: %v", t.Name(), err)
		}
		var fallbackErr error
		response, fallbackErr = fallbackHttpRequest(input)
		if fallbackErr == nil {
			err = nil
		} else {
			err = fmt.Errorf("%v: ALSO: %v", err, fallbackErr)
		}
	}

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return response, err
}

func fallbackHttpRequest(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating fallback request: %v", err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making fallback request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading fallback request: %v", err)
	}

	return string(body), err
}
