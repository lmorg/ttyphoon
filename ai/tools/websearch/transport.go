package websearch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type searchRequestError struct {
	cause   error
	message string
}

func (err *searchRequestError) Error() string { return err.message }
func (err *searchRequestError) Unwrap() error { return err.cause }

func (t *DuckDuckGoSearch) doSearXNG(request *http.Request) (*http.Response, error) {
	release := func() {}
	if t.gate != nil {
		var err error
		release, err = t.gate.acquire(request.Context())
		if err != nil {
			return nil, err
		}
	}
	for name, values := range t.headers {
		request.Header[name] = append([]string(nil), values...)
	}
	request.Header.Set("User-Agent", defaultUserAgent)
	request.Header.Set("Accept", "application/json")
	response, err := t.client.Do(request)
	if err == nil {
		if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusServiceUnavailable {
			deadline := retryAfterDeadline(response.Header.Get("Retry-After"), requestGateNow(t.gate))
			if t.gate != nil {
				t.gate.cooldown(response.StatusCode, deadline)
			}
			_ = response.Body.Close()
			release()
			return nil, rateLimitMessage(response.StatusCode, deadline, requestGateNow(t.gate))
		}
		response.Body = &requestGateBody{ReadCloser: response.Body, release: release}
		return response, nil
	}
	release()
	if request.Context().Err() != nil {
		return nil, request.Context().Err()
	}
	var requestError *url.Error
	if errors.As(err, &requestError) {
		err = requestError.Err
	}
	return nil, &searchRequestError{cause: err, message: "SearXNG request failed: " + t.redactSecrets(err.Error())}
}

func (t *DuckDuckGoSearch) redactSecrets(text string) string {
	var secrets []string
	for _, values := range t.headers {
		for _, value := range values {
			if value != "" {
				secrets = append(secrets, value)
				if parts := strings.SplitN(value, " ", 2); len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
					secrets = append(secrets, strings.TrimSpace(parts[1]))
				}
			}
		}
	}
	request := &http.Request{Header: t.headers}
	if username, password, ok := request.BasicAuth(); ok {
		if username != "" {
			secrets = append(secrets, username)
		}
		if password != "" {
			secrets = append(secrets, password)
		}
	}
	for _, cookie := range request.Cookies() {
		if cookie.Value != "" {
			secrets = append(secrets, cookie.Value)
		}
	}
	sort.Slice(secrets, func(left, right int) bool { return len(secrets[left]) > len(secrets[right]) })
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, "[REDACTED]")
		encoded, err := json.Marshal(secret)
		if err == nil && len(encoded) > 2 {
			text = strings.ReplaceAll(text, string(encoded[1:len(encoded)-1]), "[REDACTED]")
		}
	}
	return text
}
