package swagger

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lmorg/ttyphoon/app"
)

var userAgent = app.Name() + "/" + app.Version()

// Execute sends the HTTP request described by req and returns the response.
// ctx can be used to enforce a deadline or cancellation from the caller.
func Execute(ctx context.Context, req RequestT) ResponseT {
	if req.Method == "" {
		return ResponseT{Error: "request method is empty"}
	}
	if req.URL == "" {
		return ResponseT{Error: "request URL is empty"}
	}

	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, strings.ToUpper(req.Method), req.URL, bodyReader)
	if err != nil {
		return ResponseT{Error: fmt.Sprintf("building request: %v", err)}
	}

	httpReq.Header.Set("User-Agent", userAgent)
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := defaultClient.Do(httpReq)
	if err != nil {
		return ResponseT{Error: fmt.Sprintf("executing request: %v", err)}
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResponseT{Error: fmt.Sprintf("reading response body: %v", err)}
	}

	flatHeaders := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		flatHeaders[k] = resp.Header.Get(k)
	}

	return ResponseT{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    flatHeaders,
		Body:       string(rawBody),
	}
}
