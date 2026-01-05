package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LoggingTransport implements http.RoundTripper and logs requests and responses
type LoggingTransport struct {
	Transport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction and logs the request and response
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log Request
	reqBodyLog := "empty"
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body
		if len(bodyBytes) > 0 {
			reqBodyLog = string(bodyBytes)
		}
	}
	fmt.Printf("[HTTP Request] %s %s | Headers: %v | Body: %s\n", req.Method, req.URL, req.Header, reqBodyLog)

	start := time.Now()

	// Execute Request
	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("[HTTP Error] %s %s | Duration: %v | Error: %v\n", req.Method, req.URL, duration, err)
		return nil, err
	}

	// Log Response
	respBodyLog := "empty"
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body
		if len(bodyBytes) > 0 {
			// Limit log size if needed, but user asked for record, so keep full or reasonable limit
			if len(bodyBytes) > 2000 {
				respBodyLog = string(bodyBytes[:2000]) + "...(truncated)"
			} else {
				respBodyLog = string(bodyBytes)
			}
		}
	}

	fmt.Printf("[HTTP Response] %s %s | Status: %s | Duration: %v | Body: %s\n", req.Method, req.URL, resp.Status, duration, respBodyLog)

	return resp, nil
}

// NewHTTPClient returns a new http.Client with logging enabled
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &LoggingTransport{
			Transport: http.DefaultTransport,
		},
	}
}
