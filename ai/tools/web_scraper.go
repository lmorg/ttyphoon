package tools

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/chromedp/chromedp"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
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
The input for this tool is a URL to the web page.
The output of this tool will either be HTML or markdown.`
}

func (t *ChromeScraper) Name() string { return "Web Scraper" }
func (t *ChromeScraper) Path() string { return "internal" }

var (
	// I know you shouldn't use regex to parse HTML.
	// This is only used in the extreme edge case that a markdown document
	// cannot be automatically generated from the HTML document. At that
	// point the HTML parser has already failed and we are now looking to
	// use an LLM for parsing. In that instance, our token count will be
	// massive so stripping the following HTML tags via regexp, while crude,
	// will reduce the token count.
	rxHtml = []*regexp.Regexp{
		regexp.MustCompile(`(?si)<head( |>).*?</head>`),
		regexp.MustCompile(`(?si)<svg( |>).*?</svg>`),
		regexp.MustCompile(`(?si)<script( |>).*?</script>`),
	}
)

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

	t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%s: using Chrome: %s", t.meta.ServiceName(), input))

	chromeCtx, cancel := chromedp.NewContext(ctx)
	defer cancel()

	var body, article string

	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(input),
		chromedp.Sleep(3*time.Second), // this is a kludge to allow dynamic sites which require JS to finish rendering
		chromedp.InnerHTML("body", &body, chromedp.ByQuery),
		chromedp.InnerHTML("article", &article, chromedp.ByQuery),
	)

	if article != "" {
		response = article
	} else {
		response = body
	}

	if err != nil {
		t.meta.Renderer.DisplayNotification(types.NOTIFY_WARN, fmt.Sprintf("%s: couldn't start Chrome: %v", t.meta.ServiceName(), err))
		t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%s: using fallback raw HTTP: %s", t.meta.ServiceName(), input))

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

	md, mdErr := htmltomarkdown.ConvertString(response)
	if mdErr == nil {
		response = md
	} else {
		// we cannot parse the HTML document via correct methods,
		// so now lets focus on reducing the token count so the LLM
		// can parse the HTML document fast and cost-effectively.
		for _, rx := range rxHtml {
			found := rx.FindAllString(response, -1)
			for i := range found {
				log.Println(found[i])
				response = strings.Replace(response, found[i], "", 1)
			}
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
