package workbench

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DebugEntryType represents the type of debug entry.
type DebugEntryType string

const (
	DebugEntryTypeLog         DebugEntryType = "log"
	DebugEntryTypeRequest     DebugEntryType = "request"
	DebugEntryTypeResponse    DebugEntryType = "response"
	DebugEntryTypePluginCall  DebugEntryType = "plugin_call"
	DebugEntryTypePluginResult DebugEntryType = "plugin_result"
	DebugEntryTypeError       DebugEntryType = "error"
	DebugEntryTypeStateChange DebugEntryType = "state_change"
)

// DebugEntry represents a single debug entry.
type DebugEntry struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Type      DebugEntryType  `json:"type"`
	Operation string          `json:"operation"`
	Message   string          `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Duration  time.Duration   `json:"duration,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// DebugStore stores debug information for inspection.
type DebugStore struct {
	mu         sync.RWMutex
	entries    []DebugEntry
	maxEntries int
	idCounter  atomic.Int64

	// HTTP request/response tracking
	httpRequests []HTTPRequestEntry

	// Plugin call tracking (for detailed introspection)
	pluginCalls []PluginCallEntry
}

// HTTPRequestEntry tracks an HTTP request made by the connector.
type HTTPRequestEntry struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	RequestHeaders  map[string]string `json:"request_headers,omitempty"`
	RequestBody     string            `json:"request_body,omitempty"`
	ResponseStatus  int               `json:"response_status,omitempty"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty"`
	ResponseBody    string            `json:"response_body,omitempty"`
	Duration        time.Duration     `json:"duration"`
	Error           string            `json:"error,omitempty"`
}

// PluginCallEntry tracks a plugin method call.
type PluginCallEntry struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Method    string          `json:"method"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output,omitempty"`
	Duration  time.Duration   `json:"duration"`
	Error     string          `json:"error,omitempty"`
}

// NewDebugStore creates a new debug store.
func NewDebugStore(maxEntries int) *DebugStore {
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	return &DebugStore{
		entries:      make([]DebugEntry, 0, maxEntries),
		httpRequests: make([]HTTPRequestEntry, 0, maxEntries),
		pluginCalls:  make([]PluginCallEntry, 0, maxEntries),
		maxEntries:   maxEntries,
	}
}

// nextID generates the next unique ID.
func (d *DebugStore) nextID() string {
	id := d.idCounter.Add(1)
	return fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), id)
}

// Log adds a log entry.
func (d *DebugStore) Log(operation, message string) {
	d.addEntry(DebugEntry{
		Type:      DebugEntryTypeLog,
		Operation: operation,
		Message:   message,
	})
}

// LogError adds an error entry.
func (d *DebugStore) LogError(operation string, err error) {
	d.addEntry(DebugEntry{
		Type:      DebugEntryTypeError,
		Operation: operation,
		Error:     err.Error(),
	})
}

// LogPluginCall logs a plugin method call start.
func (d *DebugStore) LogPluginCall(method string, input interface{}) string {
	inputJSON, _ := json.Marshal(input)
	
	entry := PluginCallEntry{
		ID:        d.nextID(),
		Timestamp: time.Now(),
		Method:    method,
		Input:     inputJSON,
	}

	d.mu.Lock()
	d.pluginCalls = append(d.pluginCalls, entry)
	if len(d.pluginCalls) > d.maxEntries {
		d.pluginCalls = d.pluginCalls[1:]
	}
	d.mu.Unlock()

	d.addEntry(DebugEntry{
		Type:      DebugEntryTypePluginCall,
		Operation: method,
		Data:      inputJSON,
	})

	return entry.ID
}

// LogPluginResult logs a plugin method call result.
func (d *DebugStore) LogPluginResult(callID string, output interface{}, duration time.Duration, err error) {
	outputJSON, _ := json.Marshal(output)
	
	d.mu.Lock()
	for i := range d.pluginCalls {
		if d.pluginCalls[i].ID == callID {
			d.pluginCalls[i].Output = outputJSON
			d.pluginCalls[i].Duration = duration
			if err != nil {
				d.pluginCalls[i].Error = err.Error()
			}
			break
		}
	}
	d.mu.Unlock()

	entry := DebugEntry{
		Type:     DebugEntryTypePluginResult,
		Data:     outputJSON,
		Duration: duration,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	d.addEntry(entry)
}

// LogStateChange logs a state change.
func (d *DebugStore) LogStateChange(key string, oldState, newState json.RawMessage) {
	data := map[string]interface{}{
		"key":       key,
		"old_state": oldState,
		"new_state": newState,
	}
	dataJSON, _ := json.Marshal(data)

	d.addEntry(DebugEntry{
		Type:      DebugEntryTypeStateChange,
		Operation: "state_change",
		Message:   key,
		Data:      dataJSON,
	})
}

// LogHTTPRequest logs an HTTP request.
func (d *DebugStore) LogHTTPRequest(entry HTTPRequestEntry) {
	entry.ID = d.nextID()
	entry.Timestamp = time.Now()

	d.mu.Lock()
	d.httpRequests = append(d.httpRequests, entry)
	if len(d.httpRequests) > d.maxEntries {
		d.httpRequests = d.httpRequests[1:]
	}
	d.mu.Unlock()

	// Also add to general entries
	data, _ := json.Marshal(entry)
	d.addEntry(DebugEntry{
		Type:      DebugEntryTypeRequest,
		Operation: entry.Method + " " + entry.URL,
		Duration:  entry.Duration,
		Data:      data,
	})
}

func (d *DebugStore) addEntry(entry DebugEntry) {
	entry.ID = d.nextID()
	entry.Timestamp = time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	d.entries = append(d.entries, entry)
	if len(d.entries) > d.maxEntries {
		d.entries = d.entries[1:]
	}
}

// GetEntries returns all entries, optionally filtered by type.
func (d *DebugStore) GetEntries(entryType *DebugEntryType, limit int) []DebugEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit > len(d.entries) {
		limit = len(d.entries)
	}

	if entryType == nil {
		// Return last N entries (most recent first)
		result := make([]DebugEntry, limit)
		for i := 0; i < limit; i++ {
			result[i] = d.entries[len(d.entries)-1-i]
		}
		return result
	}

	// Filter by type
	var filtered []DebugEntry
	for i := len(d.entries) - 1; i >= 0 && len(filtered) < limit; i-- {
		if d.entries[i].Type == *entryType {
			filtered = append(filtered, d.entries[i])
		}
	}
	return filtered
}

// GetHTTPRequests returns HTTP request entries.
func (d *DebugStore) GetHTTPRequests(limit int) []HTTPRequestEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit > len(d.httpRequests) {
		limit = len(d.httpRequests)
	}

	// Return most recent first
	result := make([]HTTPRequestEntry, limit)
	for i := 0; i < limit; i++ {
		result[i] = d.httpRequests[len(d.httpRequests)-1-i]
	}
	return result
}

// GetHTTPRequestByID returns a specific HTTP request by ID.
func (d *DebugStore) GetHTTPRequestByID(id string) *HTTPRequestEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for i := range d.httpRequests {
		if d.httpRequests[i].ID == id {
			entry := d.httpRequests[i]
			return &entry
		}
	}
	return nil
}

// GetPluginCalls returns plugin call entries.
func (d *DebugStore) GetPluginCalls(limit int) []PluginCallEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if limit <= 0 || limit > len(d.pluginCalls) {
		limit = len(d.pluginCalls)
	}

	// Return most recent first
	result := make([]PluginCallEntry, limit)
	for i := 0; i < limit; i++ {
		result[i] = d.pluginCalls[len(d.pluginCalls)-1-i]
	}
	return result
}

// Clear clears all debug entries.
func (d *DebugStore) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.entries = make([]DebugEntry, 0, d.maxEntries)
	d.httpRequests = make([]HTTPRequestEntry, 0, d.maxEntries)
	d.pluginCalls = make([]PluginCallEntry, 0, d.maxEntries)
}

// Stats returns debug store statistics.
type DebugStats struct {
	TotalEntries    int `json:"total_entries"`
	TotalHTTPReqs   int `json:"total_http_requests"`
	TotalPluginCalls int `json:"total_plugin_calls"`
	ErrorCount      int `json:"error_count"`
}

func (d *DebugStore) Stats() DebugStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	errorCount := 0
	for _, e := range d.entries {
		if e.Type == DebugEntryTypeError || e.Error != "" {
			errorCount++
		}
	}

	return DebugStats{
		TotalEntries:     len(d.entries),
		TotalHTTPReqs:    len(d.httpRequests),
		TotalPluginCalls: len(d.pluginCalls),
		ErrorCount:       errorCount,
	}
}
