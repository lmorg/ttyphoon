package tools

import (
	"context"
	"log"

	"github.com/chromedp/chromedp"
	"github.com/lmorg/murex/utils/which"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/tmc/langchaingo/callbacks"
)

var _CHROME_INSTALLED = which.Which("chrome") != "" || which.Which("chromium") != ""

type ChromeScraper struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	enabled          bool
}

func init() {
	agent.ToolsAdd(&ChromeScraper{})
}

func (t ChromeScraper) New(meta *agent.Meta) (agent.Tool, error) {
	return &ChromeScraper{meta: meta, enabled: _CHROME_INSTALLED}, nil
}

func (t *ChromeScraper) Enabled() bool { return t.enabled }
func (t *ChromeScraper) Toggle()       { t.enabled = !t.enabled }

func (t *ChromeScraper) Description() string {
	return `Loads a web page in Chrome and returns the contents of its page.
Useful for checking online content.
The input for this tool is a URL to the web page.`
}

func (t *ChromeScraper) Name() string { return "Chrome Scraper" }
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

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, response)
	}

	return response, err
}
