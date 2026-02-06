package workbench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Snapshot represents a captured HTTP request/response pair for testing.
type Snapshot struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	Tags        []string  `json:"tags,omitempty"`

	// Source info
	Provider    string `json:"provider"`
	Operation   string `json:"operation"` // e.g., "fetch_accounts", "fetch_payments"
	
	// The captured HTTP exchange
	Request  SnapshotRequest  `json:"request"`
	Response SnapshotResponse `json:"response"`

	// Original capture ID (for reference)
	CaptureID string `json:"capture_id,omitempty"`
}

// SnapshotRequest represents the HTTP request part of a snapshot.
type SnapshotRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// SnapshotResponse represents the HTTP response part of a snapshot.
type SnapshotResponse struct {
	StatusCode int               `json:"status_code"`
	Status     string            `json:"status,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
}

// SnapshotGroup groups snapshots by operation for test generation.
type SnapshotGroup struct {
	Operation string     `json:"operation"`
	Snapshots []Snapshot `json:"snapshots"`
}

// SnapshotManager manages test snapshots.
type SnapshotManager struct {
	mu sync.RWMutex

	// In-memory storage
	snapshots map[string]*Snapshot

	// Provider name
	provider string

	// Base directory for saving snapshots
	baseDir string

	// Debug store for accessing captured requests
	debug *DebugStore
}

// NewSnapshotManager creates a new snapshot manager.
func NewSnapshotManager(provider string, debug *DebugStore) *SnapshotManager {
	return &SnapshotManager{
		snapshots: make(map[string]*Snapshot),
		provider:  provider,
		debug:     debug,
	}
}

// SetBaseDir sets the base directory for saving snapshots.
func (m *SnapshotManager) SetBaseDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.baseDir = dir
}

// SaveFromCapture creates a snapshot from a captured HTTP request.
func (m *SnapshotManager) SaveFromCapture(captureID string, name string, operation string, description string, tags []string) (*Snapshot, error) {
	captured := m.debug.GetHTTPRequestByID(captureID)
	if captured == nil {
		return nil, fmt.Errorf("capture not found: %s", captureID)
	}

	snapshot := &Snapshot{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Tags:        tags,
		Provider:    m.provider,
		Operation:   operation,
		CaptureID:   captureID,
		Request: SnapshotRequest{
			Method:  captured.Method,
			URL:     captured.URL,
			Headers: captured.RequestHeaders,
			Body:    captured.RequestBody,
		},
		Response: SnapshotResponse{
			StatusCode: captured.ResponseStatus,
			Headers:    captured.ResponseHeaders,
			Body:       captured.ResponseBody,
		},
	}

	m.mu.Lock()
	m.snapshots[snapshot.ID] = snapshot
	m.mu.Unlock()

	return snapshot, nil
}

// Save adds a snapshot directly.
func (m *SnapshotManager) Save(snapshot *Snapshot) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.New().String()
	}
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = time.Now()
	}
	if snapshot.Provider == "" {
		snapshot.Provider = m.provider
	}

	m.mu.Lock()
	m.snapshots[snapshot.ID] = snapshot
	m.mu.Unlock()

	return nil
}

// Get returns a snapshot by ID.
func (m *SnapshotManager) Get(id string) *Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snapshots[id]
}

// List returns all snapshots, optionally filtered.
func (m *SnapshotManager) List(operation string, tags []string) []*Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Snapshot
	for _, s := range m.snapshots {
		// Filter by operation
		if operation != "" && s.Operation != operation {
			continue
		}
		// Filter by tags (all tags must match)
		if len(tags) > 0 {
			match := true
			for _, tag := range tags {
				found := false
				for _, st := range s.Tags {
					if st == tag {
						found = true
						break
					}
				}
				if !found {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		result = append(result, s)
	}

	// Sort by creation time (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// Delete removes a snapshot.
func (m *SnapshotManager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.snapshots[id]; !exists {
		return fmt.Errorf("snapshot not found: %s", id)
	}
	delete(m.snapshots, id)
	return nil
}

// GroupByOperation groups snapshots by operation.
func (m *SnapshotManager) GroupByOperation() []SnapshotGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := make(map[string][]Snapshot)
	for _, s := range m.snapshots {
		groups[s.Operation] = append(groups[s.Operation], *s)
	}

	var result []SnapshotGroup
	for op, snapshots := range groups {
		result = append(result, SnapshotGroup{
			Operation: op,
			Snapshots: snapshots,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Operation < result[j].Operation
	})

	return result
}

// ExportToDir exports all snapshots to a directory as JSON files.
func (m *SnapshotManager) ExportToDir(dir string) error {
	m.mu.RLock()
	snapshots := make([]*Snapshot, 0, len(m.snapshots))
	for _, s := range m.snapshots {
		snapshots = append(snapshots, s)
	}
	m.mu.RUnlock()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, s := range snapshots {
		filename := sanitizeFilename(s.Name) + ".json"
		path := filepath.Join(dir, filename)

		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot %s: %w", s.ID, err)
		}

		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write snapshot %s: %w", s.ID, err)
		}
	}

	return nil
}

// ImportFromDir imports snapshots from a directory.
func (m *SnapshotManager) ImportFromDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var snapshot Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		_ = m.Save(&snapshot)
		count++
	}

	return count, nil
}

// Clear removes all snapshots.
func (m *SnapshotManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots = make(map[string]*Snapshot)
}

// Stats returns snapshot statistics.
type SnapshotStats struct {
	Total        int            `json:"total"`
	ByOperation  map[string]int `json:"by_operation"`
	ByTag        map[string]int `json:"by_tag"`
}

func (m *SnapshotManager) Stats() SnapshotStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := SnapshotStats{
		Total:       len(m.snapshots),
		ByOperation: make(map[string]int),
		ByTag:       make(map[string]int),
	}

	for _, s := range m.snapshots {
		stats.ByOperation[s.Operation]++
		for _, tag := range s.Tags {
			stats.ByTag[tag]++
		}
	}

	return stats
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(strings.ToLower(name))
}
