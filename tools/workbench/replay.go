package workbench

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ReplayRequest represents a request to be replayed (possibly modified).
type ReplayRequest struct {
	// Original request ID (if replaying an existing request)
	OriginalID string `json:"original_id,omitempty"`

	// Request details (can be modified from original)
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// ReplayResponse represents the response from a replayed request.
type ReplayResponse struct {
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`

	// Original request info (if this was a replay)
	OriginalID string `json:"original_id,omitempty"`

	// Request that was sent
	Request ReplayRequest `json:"request"`

	// Response received
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`

	// Error if request failed
	Error string `json:"error,omitempty"`
}

// Replayer handles replaying HTTP requests.
type Replayer struct {
	mu sync.RWMutex

	// Client to use for replay requests (doesn't go through debug transport)
	client *http.Client

	// History of replays
	history []ReplayResponse

	// Max history size
	maxHistory int

	// Debug store to look up original requests
	debug *DebugStore
}

// NewReplayer creates a new replayer.
func NewReplayer(debug *DebugStore) *Replayer {
	return &Replayer{
		client: &http.Client{
			Timeout: 30 * time.Second,
			// Use a fresh transport that doesn't go through our debug capture
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
			},
		},
		history:    make([]ReplayResponse, 0),
		maxHistory: 100,
		debug:      debug,
	}
}

// Replay executes a request and returns the response.
func (r *Replayer) Replay(ctx context.Context, req ReplayRequest) (*ReplayResponse, error) {
	// Validate request
	if req.Method == "" {
		return nil, fmt.Errorf("method is required")
	}
	if req.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	// Create HTTP request
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request
	start := time.Now()
	httpResp, err := r.client.Do(httpReq)
	duration := time.Since(start)

	response := &ReplayResponse{
		ID:         uuid.New().String(),
		Timestamp:  time.Now(),
		Duration:   duration,
		OriginalID: req.OriginalID,
		Request:    req,
	}

	if err != nil {
		response.Error = err.Error()
		r.addToHistory(*response)
		return response, nil // Return response with error, not error
	}
	defer httpResp.Body.Close()

	// Capture response
	response.StatusCode = httpResp.StatusCode
	response.Status = httpResp.Status
	response.Headers = make(map[string]string)
	for key, values := range httpResp.Header {
		response.Headers[key] = strings.Join(values, ", ")
	}

	// Read body
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		response.Error = fmt.Sprintf("failed to read response body: %v", err)
	} else {
		// Limit body size
		if len(bodyBytes) > 256*1024 {
			response.Body = string(bodyBytes[:256*1024]) + "\n... [truncated]"
		} else {
			response.Body = string(bodyBytes)
		}
	}

	r.addToHistory(*response)
	return response, nil
}

// ReplayFromCapture replays a captured request by ID, optionally with modifications.
func (r *Replayer) ReplayFromCapture(ctx context.Context, captureID string, modifications *ReplayRequest) (*ReplayResponse, error) {
	// Find the original request
	original := r.debug.GetHTTPRequestByID(captureID)
	if original == nil {
		return nil, fmt.Errorf("request not found: %s", captureID)
	}

	// Build replay request from original
	req := ReplayRequest{
		OriginalID: captureID,
		Method:     original.Method,
		URL:        original.URL,
		Headers:    copyHeaders(original.RequestHeaders),
		Body:       original.RequestBody,
	}

	// Apply modifications if provided
	if modifications != nil {
		if modifications.Method != "" {
			req.Method = modifications.Method
		}
		if modifications.URL != "" {
			req.URL = modifications.URL
		}
		if modifications.Headers != nil {
			for k, v := range modifications.Headers {
				if v == "" {
					delete(req.Headers, k)
				} else {
					req.Headers[k] = v
				}
			}
		}
		if modifications.Body != "" {
			req.Body = modifications.Body
		}
	}

	return r.Replay(ctx, req)
}

func (r *Replayer) addToHistory(resp ReplayResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history = append(r.history, resp)
	if len(r.history) > r.maxHistory {
		r.history = r.history[1:]
	}
}

// GetHistory returns the replay history.
func (r *Replayer) GetHistory(limit int) []ReplayResponse {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.history) {
		limit = len(r.history)
	}

	// Return most recent first
	result := make([]ReplayResponse, limit)
	for i := 0; i < limit; i++ {
		result[i] = r.history[len(r.history)-1-i]
	}
	return result
}

// GetReplayByID returns a specific replay response.
func (r *Replayer) GetReplayByID(id string) *ReplayResponse {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.history {
		if r.history[i].ID == id {
			return &r.history[i]
		}
	}
	return nil
}

// ClearHistory clears replay history.
func (r *Replayer) ClearHistory() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = make([]ReplayResponse, 0)
}

// CompareResponses compares original and replayed responses.
type ResponseComparison struct {
	OriginalID   string `json:"original_id"`
	ReplayID     string `json:"replay_id"`
	
	// Status comparison
	StatusMatch  bool   `json:"status_match"`
	OriginalStatus int  `json:"original_status"`
	ReplayStatus   int  `json:"replay_status"`

	// Body comparison
	BodyMatch    bool   `json:"body_match"`
	BodyDiff     string `json:"body_diff,omitempty"`

	// Header differences
	HeaderDiffs  []HeaderDiff `json:"header_diffs,omitempty"`
}

type HeaderDiff struct {
	Key          string `json:"key"`
	OriginalValue string `json:"original_value,omitempty"`
	ReplayValue   string `json:"replay_value,omitempty"`
	Type          string `json:"type"` // "added", "removed", "changed"
}

// Compare compares a replay response with the original captured request.
func (r *Replayer) Compare(replayID string) (*ResponseComparison, error) {
	replay := r.GetReplayByID(replayID)
	if replay == nil {
		return nil, fmt.Errorf("replay not found: %s", replayID)
	}

	if replay.OriginalID == "" {
		return nil, fmt.Errorf("replay has no original request to compare with")
	}

	original := r.debug.GetHTTPRequestByID(replay.OriginalID)
	if original == nil {
		return nil, fmt.Errorf("original request not found: %s", replay.OriginalID)
	}

	comparison := &ResponseComparison{
		OriginalID:     replay.OriginalID,
		ReplayID:       replayID,
		OriginalStatus: original.ResponseStatus,
		ReplayStatus:   replay.StatusCode,
		StatusMatch:    original.ResponseStatus == replay.StatusCode,
		BodyMatch:      original.ResponseBody == replay.Body,
	}

	// Compare headers
	allKeys := make(map[string]bool)
	for k := range original.ResponseHeaders {
		allKeys[k] = true
	}
	for k := range replay.Headers {
		allKeys[k] = true
	}

	for key := range allKeys {
		origVal := original.ResponseHeaders[key]
		replayVal := replay.Headers[key]

		if origVal == "" && replayVal != "" {
			comparison.HeaderDiffs = append(comparison.HeaderDiffs, HeaderDiff{
				Key:         key,
				ReplayValue: replayVal,
				Type:        "added",
			})
		} else if origVal != "" && replayVal == "" {
			comparison.HeaderDiffs = append(comparison.HeaderDiffs, HeaderDiff{
				Key:           key,
				OriginalValue: origVal,
				Type:          "removed",
			})
		} else if origVal != replayVal {
			comparison.HeaderDiffs = append(comparison.HeaderDiffs, HeaderDiff{
				Key:           key,
				OriginalValue: origVal,
				ReplayValue:   replayVal,
				Type:          "changed",
			})
		}
	}

	return comparison, nil
}

func copyHeaders(h map[string]string) map[string]string {
	if h == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(h))
	for k, v := range h {
		result[k] = v
	}
	return result
}

// DryRun shows what a replay would send without actually sending it.
func (r *Replayer) DryRun(ctx context.Context, req ReplayRequest) (*ReplayRequest, error) {
	// If replaying from capture, resolve the original
	if req.OriginalID != "" && req.Method == "" {
		original := r.debug.GetHTTPRequestByID(req.OriginalID)
		if original == nil {
			return nil, fmt.Errorf("request not found: %s", req.OriginalID)
		}
		
		result := &ReplayRequest{
			OriginalID: req.OriginalID,
			Method:     original.Method,
			URL:        original.URL,
			Headers:    copyHeaders(original.RequestHeaders),
			Body:       original.RequestBody,
		}
		return result, nil
	}

	return &req, nil
}

// CreateCurlCommand generates a curl command for a request.
func (r *Replayer) CreateCurlCommand(req ReplayRequest) string {
	var b bytes.Buffer
	b.WriteString("curl -X ")
	b.WriteString(req.Method)
	
	for key, value := range req.Headers {
		b.WriteString(" \\\n  -H '")
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(strings.ReplaceAll(value, "'", "'\\''"))
		b.WriteString("'")
	}

	if req.Body != "" {
		b.WriteString(" \\\n  -d '")
		b.WriteString(strings.ReplaceAll(req.Body, "'", "'\\''"))
		b.WriteString("'")
	}

	b.WriteString(" \\\n  '")
	b.WriteString(req.URL)
	b.WriteString("'")

	return b.String()
}
