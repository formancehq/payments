package workbench

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// DebugTransport is an http.RoundTripper that captures all HTTP traffic
// for debugging purposes.
type DebugTransport struct {
	// Base is the underlying transport. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// Debug is the debug store to record requests to
	Debug *DebugStore

	// Schemas is the schema manager for auto-inference (optional)
	Schemas *SchemaManager

	// MaxBodySize is the maximum size of request/response bodies to capture.
	// Bodies larger than this will be truncated. Default is 64KB.
	MaxBodySize int

	// Enabled controls whether capture is active
	enabled atomic.Bool

	// SensitiveHeaders are headers that should be redacted in logs
	SensitiveHeaders []string
}

// NewDebugTransport creates a new debug transport.
func NewDebugTransport(debug *DebugStore) *DebugTransport {
	t := &DebugTransport{
		Base:        http.DefaultTransport,
		Debug:       debug,
		MaxBodySize: 64 * 1024, // 64KB
		SensitiveHeaders: []string{
			"Authorization",
			"X-Api-Key",
			"X-API-Key",
			"Api-Key",
			"Apikey",
			"Cookie",
			"Set-Cookie",
			"X-Auth-Token",
			"X-Access-Token",
		},
	}
	t.enabled.Store(true)
	return t
}

// Enable enables HTTP capture.
func (t *DebugTransport) Enable() {
	t.enabled.Store(true)
}

// Disable disables HTTP capture.
func (t *DebugTransport) Disable() {
	t.enabled.Store(false)
}

// IsEnabled returns whether capture is enabled.
func (t *DebugTransport) IsEnabled() bool {
	return t.enabled.Load()
}

// RoundTrip implements http.RoundTripper.
func (t *DebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !t.enabled.Load() || t.Debug == nil {
		return t.base().RoundTrip(req)
	}

	entry := HTTPRequestEntry{
		Timestamp: time.Now(),
		Method:    req.Method,
		URL:       req.URL.String(),
	}

	// Capture request headers (redacting sensitive ones)
	entry.RequestHeaders = t.captureHeaders(req.Header)

	// Capture request body
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			req.Body.Close()
			// Restore the body for the actual request
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			entry.RequestBody = t.truncateBody(bodyBytes)
		}
	}

	// Make the actual request
	start := time.Now()
	resp, err := t.base().RoundTrip(req)
	entry.Duration = time.Since(start)

	if err != nil {
		entry.Error = err.Error()
		t.Debug.LogHTTPRequest(entry)
		return nil, err
	}

	// Capture response
	entry.ResponseStatus = resp.StatusCode
	entry.ResponseHeaders = t.captureHeaders(resp.Header)

	// Capture response body
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			resp.Body.Close()
			// Restore the body for the caller
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			entry.ResponseBody = t.truncateBody(bodyBytes)
		}
	}

	t.Debug.LogHTTPRequest(entry)

	// Auto-infer schema from JSON responses
	if t.Schemas != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") && entry.ResponseBody != "" {
			// Extract operation from URL path
			operation := extractOperationFromURL(req.URL.Path)
			_, _ = t.Schemas.InferFromJSON(operation, req.URL.Path, req.Method, []byte(entry.ResponseBody))
		}
	}

	return resp, nil
}

// extractOperationFromURL extracts a reasonable operation name from a URL path.
func extractOperationFromURL(path string) string {
	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	// Try to find meaningful segment (skip version numbers, IDs)
	for _, part := range parts {
		// Skip version prefixes like v1, v2
		if len(part) == 2 && part[0] == 'v' && part[1] >= '0' && part[1] <= '9' {
			continue
		}
		// Skip empty or very short segments
		if len(part) < 2 {
			continue
		}
		// Skip segments that look like IDs (all hex, numbers, or UUIDs)
		if looksLikeID(part) {
			continue
		}
		return part
	}

	// Fallback to full path
	return strings.ReplaceAll(path, "/", "_")
}

// looksLikeID checks if a string looks like an ID (UUID, hex, numeric).
func looksLikeID(s string) bool {
	if len(s) > 30 {
		return true // Long strings are likely IDs
	}

	// Check if it's all hex characters (with optional dashes for UUIDs)
	allHex := true
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-' || c == '_') {
			allHex = false
			break
		}
	}
	if allHex && len(s) >= 8 {
		return true
	}

	return false
}

func (t *DebugTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *DebugTransport) captureHeaders(h http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range h {
		if t.isSensitiveHeader(key) {
			result[key] = "[REDACTED]"
		} else {
			result[key] = strings.Join(values, ", ")
		}
	}
	return result
}

func (t *DebugTransport) isSensitiveHeader(key string) bool {
	keyLower := strings.ToLower(key)
	for _, sensitive := range t.SensitiveHeaders {
		if strings.ToLower(sensitive) == keyLower {
			return true
		}
	}
	return false
}

func (t *DebugTransport) truncateBody(body []byte) string {
	if len(body) > t.MaxBodySize {
		return string(body[:t.MaxBodySize]) + "\n... [truncated]"
	}
	return string(body)
}

// InstallGlobalTransport installs the debug transport as the default HTTP transport.
// This affects all HTTP clients that use http.DefaultTransport.
// Returns the previous default transport so it can be restored later.
func InstallGlobalTransport(debug *DebugStore) (*DebugTransport, http.RoundTripper) {
	previous := http.DefaultTransport
	transport := NewDebugTransport(debug)
	transport.Base = previous
	http.DefaultTransport = transport
	return transport, previous
}

// RestoreGlobalTransport restores the default HTTP transport.
func RestoreGlobalTransport(original http.RoundTripper) {
	http.DefaultTransport = original
}

// WrapTransport wraps an existing transport with debug capture.
func WrapTransport(base http.RoundTripper, debug *DebugStore) *DebugTransport {
	transport := NewDebugTransport(debug)
	transport.Base = base
	return transport
}
