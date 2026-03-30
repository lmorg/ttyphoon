// Package swagger provides HTTP request execution for Swagger/OpenAPI endpoints.
// It is intentionally backend-only: no parsing of JSON specs, no frontend logic.
package swagger

import "net/http"

// RequestT describes a single API call to execute.
type RequestT struct {
	// Method is the HTTP verb, e.g. "GET", "POST".
	Method string `json:"method"`
	// URL is the fully-qualified URL to call.
	URL string `json:"url"`
	// Headers is a map of request header names to values.
	Headers map[string]string `json:"headers"`
	// Body is the raw request body (JSON string, or empty).
	Body string `json:"body"`
}

// ResponseT is the result returned to the frontend.
type ResponseT struct {
	// StatusCode is the numeric HTTP status, e.g. 200.
	StatusCode int `json:"statusCode"`
	// Status is the human-readable status line, e.g. "200 OK".
	Status string `json:"status"`
	// Headers is a flat map of response header names to their (first) value.
	Headers map[string]string `json:"headers"`
	// Body is the raw response body text.
	Body string `json:"body"`
	// Error is non-empty when the request could not be made at all.
	Error string `json:"error,omitempty"`
}

// defaultClient is shared across calls so connections are reused.
var defaultClient = &http.Client{Transport: &http.Transport{}}
