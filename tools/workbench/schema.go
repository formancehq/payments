package workbench

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// SchemaType represents the inferred type of a JSON field.
type SchemaType string

const (
	SchemaTypeString  SchemaType = "string"
	SchemaTypeNumber  SchemaType = "number"
	SchemaTypeBoolean SchemaType = "boolean"
	SchemaTypeNull    SchemaType = "null"
	SchemaTypeArray   SchemaType = "array"
	SchemaTypeObject  SchemaType = "object"
	SchemaTypeMixed   SchemaType = "mixed" // When we've seen multiple types
)

// FieldSchema represents the schema of a single field.
type FieldSchema struct {
	Name       string                 `json:"name"`
	Path       string                 `json:"path"` // Full path like "data.accounts[].id"
	Type       SchemaType             `json:"type"`
	Types      []SchemaType           `json:"types,omitempty"` // All observed types
	Required   bool                   `json:"required"`        // Always present
	Nullable   bool                   `json:"nullable"`        // Sometimes null
	ArrayItem  *FieldSchema           `json:"array_item,omitempty"`
	Properties map[string]*FieldSchema `json:"properties,omitempty"`
	Examples   []interface{}          `json:"examples,omitempty"` // Sample values
	SeenCount  int                    `json:"seen_count"`
}

// InferredSchema represents the complete schema for an API response.
type InferredSchema struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Operation   string                  `json:"operation"`
	Endpoint    string                  `json:"endpoint"` // URL pattern
	Method      string                  `json:"method"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
	SampleCount int                     `json:"sample_count"`
	Root        *FieldSchema            `json:"root"`
}

// SchemaDiff represents differences between two schemas.
type SchemaDiff struct {
	Timestamp    time.Time        `json:"timestamp"`
	BaselineID   string           `json:"baseline_id"`
	CurrentID    string           `json:"current_id,omitempty"`
	Operation    string           `json:"operation"`
	HasChanges   bool             `json:"has_changes"`
	AddedFields  []FieldChange    `json:"added_fields,omitempty"`
	RemovedFields []FieldChange   `json:"removed_fields,omitempty"`
	TypeChanges  []TypeChange     `json:"type_changes,omitempty"`
	Summary      string           `json:"summary"`
}

// FieldChange represents a field addition or removal.
type FieldChange struct {
	Path string     `json:"path"`
	Type SchemaType `json:"type"`
}

// TypeChange represents a type change for a field.
type TypeChange struct {
	Path    string     `json:"path"`
	OldType SchemaType `json:"old_type"`
	NewType SchemaType `json:"new_type"`
}

// SchemaManager manages schema inference and comparison.
type SchemaManager struct {
	mu sync.RWMutex

	// Inferred schemas by operation
	schemas map[string]*InferredSchema

	// Baseline schemas (saved for comparison)
	baselines map[string]*InferredSchema

	provider string
}

// NewSchemaManager creates a new schema manager.
func NewSchemaManager(provider string) *SchemaManager {
	return &SchemaManager{
		schemas:   make(map[string]*InferredSchema),
		baselines: make(map[string]*InferredSchema),
		provider:  provider,
	}
}

// InferFromJSON infers/updates schema from a JSON response.
func (m *SchemaManager) InferFromJSON(operation string, endpoint string, method string, jsonData []byte) (*InferredSchema, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	schema, exists := m.schemas[operation]
	if !exists {
		schema = &InferredSchema{
			ID:        fmt.Sprintf("%s-%d", operation, time.Now().UnixNano()),
			Name:      operation,
			Operation: operation,
			Endpoint:  endpoint,
			Method:    method,
			CreatedAt: time.Now(),
			Root:      &FieldSchema{Name: "root", Path: "", Properties: make(map[string]*FieldSchema)},
		}
		m.schemas[operation] = schema
	}

	// Update schema with new data
	m.inferField(schema.Root, "", data)
	schema.SampleCount++
	schema.UpdatedAt = time.Now()

	return schema, nil
}

func (m *SchemaManager) inferField(field *FieldSchema, path string, value interface{}) {
	field.SeenCount++

	if value == nil {
		field.Nullable = true
		m.addType(field, SchemaTypeNull)
		return
	}

	switch v := value.(type) {
	case bool:
		m.addType(field, SchemaTypeBoolean)
		m.addExample(field, v)

	case float64:
		m.addType(field, SchemaTypeNumber)
		m.addExample(field, v)

	case string:
		m.addType(field, SchemaTypeString)
		m.addExample(field, v)

	case []interface{}:
		m.addType(field, SchemaTypeArray)
		if field.ArrayItem == nil {
			field.ArrayItem = &FieldSchema{Name: "[]", Path: path + "[]", Properties: make(map[string]*FieldSchema)}
		}
		for _, item := range v {
			m.inferField(field.ArrayItem, path+"[]", item)
		}

	case map[string]interface{}:
		m.addType(field, SchemaTypeObject)
		if field.Properties == nil {
			field.Properties = make(map[string]*FieldSchema)
		}
		for key, val := range v {
			childPath := path
			if childPath != "" {
				childPath += "."
			}
			childPath += key

			child, exists := field.Properties[key]
			if !exists {
				child = &FieldSchema{Name: key, Path: childPath, Properties: make(map[string]*FieldSchema)}
				field.Properties[key] = child
			}
			m.inferField(child, childPath, val)
		}
	}
}

func (m *SchemaManager) addType(field *FieldSchema, t SchemaType) {
	// Check if we've already seen this type
	for _, existing := range field.Types {
		if existing == t {
			return
		}
	}
	field.Types = append(field.Types, t)

	// Update the main type
	if len(field.Types) == 1 {
		field.Type = t
	} else {
		field.Type = SchemaTypeMixed
	}
}

func (m *SchemaManager) addExample(field *FieldSchema, value interface{}) {
	// Keep up to 3 unique examples
	if len(field.Examples) >= 3 {
		return
	}
	for _, ex := range field.Examples {
		if reflect.DeepEqual(ex, value) {
			return
		}
	}
	field.Examples = append(field.Examples, value)
}

// GetSchema returns the schema for an operation.
func (m *SchemaManager) GetSchema(operation string) *InferredSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.schemas[operation]
}

// ListSchemas returns all inferred schemas.
func (m *SchemaManager) ListSchemas() []*InferredSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*InferredSchema
	for _, s := range m.schemas {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Operation < result[j].Operation
	})
	return result
}

// SaveBaseline saves the current schema as a baseline.
func (m *SchemaManager) SaveBaseline(operation string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	schema, exists := m.schemas[operation]
	if !exists {
		return fmt.Errorf("no schema for operation: %s", operation)
	}

	// Deep copy the schema
	data, _ := json.Marshal(schema)
	var baseline InferredSchema
	_ = json.Unmarshal(data, &baseline)
	baseline.ID = fmt.Sprintf("baseline-%s-%d", operation, time.Now().UnixNano())

	m.baselines[operation] = &baseline
	return nil
}

// SaveAllBaselines saves all current schemas as baselines.
func (m *SchemaManager) SaveAllBaselines() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for op, schema := range m.schemas {
		data, _ := json.Marshal(schema)
		var baseline InferredSchema
		_ = json.Unmarshal(data, &baseline)
		baseline.ID = fmt.Sprintf("baseline-%s-%d", op, time.Now().UnixNano())
		m.baselines[op] = &baseline
		count++
	}
	return count
}

// GetBaseline returns the baseline for an operation.
func (m *SchemaManager) GetBaseline(operation string) *InferredSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.baselines[operation]
}

// ListBaselines returns all baselines.
func (m *SchemaManager) ListBaselines() []*InferredSchema {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*InferredSchema
	for _, s := range m.baselines {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Operation < result[j].Operation
	})
	return result
}

// CompareWithBaseline compares current schema against baseline.
func (m *SchemaManager) CompareWithBaseline(operation string) (*SchemaDiff, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	baseline, hasBaseline := m.baselines[operation]
	if !hasBaseline {
		return nil, fmt.Errorf("no baseline for operation: %s", operation)
	}

	current, hasCurrent := m.schemas[operation]
	if !hasCurrent {
		return nil, fmt.Errorf("no current schema for operation: %s", operation)
	}

	return m.compareSchemas(baseline, current), nil
}

// CompareJSONWithBaseline compares JSON data directly against baseline.
func (m *SchemaManager) CompareJSONWithBaseline(operation string, jsonData []byte) (*SchemaDiff, error) {
	m.mu.RLock()
	baseline, hasBaseline := m.baselines[operation]
	m.mu.RUnlock()

	if !hasBaseline {
		return nil, fmt.Errorf("no baseline for operation: %s", operation)
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Create temporary schema from JSON
	tempSchema := &InferredSchema{
		Operation: operation,
		Root:      &FieldSchema{Name: "root", Path: "", Properties: make(map[string]*FieldSchema)},
	}
	m.inferField(tempSchema.Root, "", data)

	return m.compareSchemas(baseline, tempSchema), nil
}

func (m *SchemaManager) compareSchemas(baseline, current *InferredSchema) *SchemaDiff {
	diff := &SchemaDiff{
		Timestamp:  time.Now(),
		BaselineID: baseline.ID,
		CurrentID:  current.ID,
		Operation:  baseline.Operation,
	}

	// Collect all paths from both schemas
	baselinePaths := make(map[string]*FieldSchema)
	currentPaths := make(map[string]*FieldSchema)

	m.collectPaths(baseline.Root, baselinePaths)
	m.collectPaths(current.Root, currentPaths)

	// Find added fields
	for path, field := range currentPaths {
		if _, exists := baselinePaths[path]; !exists {
			diff.AddedFields = append(diff.AddedFields, FieldChange{
				Path: path,
				Type: field.Type,
			})
		}
	}

	// Find removed fields
	for path, field := range baselinePaths {
		if _, exists := currentPaths[path]; !exists {
			diff.RemovedFields = append(diff.RemovedFields, FieldChange{
				Path: path,
				Type: field.Type,
			})
		}
	}

	// Find type changes
	for path, baselineField := range baselinePaths {
		if currentField, exists := currentPaths[path]; exists {
			if baselineField.Type != currentField.Type {
				diff.TypeChanges = append(diff.TypeChanges, TypeChange{
					Path:    path,
					OldType: baselineField.Type,
					NewType: currentField.Type,
				})
			}
		}
	}

	// Sort for consistent output
	sort.Slice(diff.AddedFields, func(i, j int) bool {
		return diff.AddedFields[i].Path < diff.AddedFields[j].Path
	})
	sort.Slice(diff.RemovedFields, func(i, j int) bool {
		return diff.RemovedFields[i].Path < diff.RemovedFields[j].Path
	})
	sort.Slice(diff.TypeChanges, func(i, j int) bool {
		return diff.TypeChanges[i].Path < diff.TypeChanges[j].Path
	})

	diff.HasChanges = len(diff.AddedFields) > 0 || len(diff.RemovedFields) > 0 || len(diff.TypeChanges) > 0

	// Generate summary
	var parts []string
	if len(diff.AddedFields) > 0 {
		parts = append(parts, fmt.Sprintf("+%d fields", len(diff.AddedFields)))
	}
	if len(diff.RemovedFields) > 0 {
		parts = append(parts, fmt.Sprintf("-%d fields", len(diff.RemovedFields)))
	}
	if len(diff.TypeChanges) > 0 {
		parts = append(parts, fmt.Sprintf("~%d type changes", len(diff.TypeChanges)))
	}
	if len(parts) == 0 {
		diff.Summary = "No changes detected"
	} else {
		diff.Summary = strings.Join(parts, ", ")
	}

	return diff
}

func (m *SchemaManager) collectPaths(field *FieldSchema, paths map[string]*FieldSchema) {
	if field == nil {
		return
	}

	if field.Path != "" {
		paths[field.Path] = field
	}

	for _, child := range field.Properties {
		m.collectPaths(child, paths)
	}

	if field.ArrayItem != nil {
		m.collectPaths(field.ArrayItem, paths)
	}
}

// Clear clears all schemas and baselines.
func (m *SchemaManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schemas = make(map[string]*InferredSchema)
	m.baselines = make(map[string]*InferredSchema)
}

// ClearBaselines clears only baselines.
func (m *SchemaManager) ClearBaselines() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.baselines = make(map[string]*InferredSchema)
}

// ExportBaselines exports baselines as JSON.
func (m *SchemaManager) ExportBaselines() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.MarshalIndent(m.baselines, "", "  ")
}

// ImportBaselines imports baselines from JSON.
func (m *SchemaManager) ImportBaselines(data []byte) error {
	var baselines map[string]*InferredSchema
	if err := json.Unmarshal(data, &baselines); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.baselines = baselines
	return nil
}

// SchemaStats returns statistics about inferred schemas.
type SchemaStats struct {
	TotalSchemas   int            `json:"total_schemas"`
	TotalBaselines int            `json:"total_baselines"`
	ByOperation    map[string]int `json:"by_operation"` // sample counts
}

func (m *SchemaManager) Stats() SchemaStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := SchemaStats{
		TotalSchemas:   len(m.schemas),
		TotalBaselines: len(m.baselines),
		ByOperation:    make(map[string]int),
	}

	for op, schema := range m.schemas {
		stats.ByOperation[op] = schema.SampleCount
	}

	return stats
}
