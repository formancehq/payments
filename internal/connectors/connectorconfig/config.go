// Package connectorconfig separates readable connector metadata from opaque
// provider configuration. Stored provider fields are never public by default.
package connectorconfig

import (
	"bytes"
	"encoding/json"
	"errors"
)

// Readable returns only shared, non-secret string metadata. Invalid stored
// configurations fail closed. In particular, objects under allowed keys cannot
// smuggle nested provider credentials into a response.
func Readable(stored json.RawMessage) json.RawMessage {
	var fields map[string]json.RawMessage
	result := make(map[string]string)
	if json.Unmarshal(stored, &fields) == nil {
		for _, key := range []string{"name", "pollingPeriod"} {
			var value string
			if raw, ok := fields[key]; ok && json.Unmarshal(raw, &value) == nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				result[key] = value
			}
		}
	}
	out, _ := json.Marshal(result) // string maps always marshal successfully
	return out
}

// Merge preserves omitted fields recursively without needing to interpret
// provider-specific credentials. Explicit values, including empty strings and
// arrays and null, replace stored values and must subsequently pass provider
// validation. The provider decides whether an explicit value is valid.
func Merge(stored, update json.RawMessage) (json.RawMessage, error) {
	var current, patch map[string]json.RawMessage
	if len(stored) == 0 {
		stored = json.RawMessage(`{}`)
	}
	if json.Unmarshal(stored, &current) != nil || current == nil {
		return nil, errors.New("invalid stored connector configuration")
	}
	if json.Unmarshal(update, &patch) != nil || patch == nil {
		return nil, errors.New("connector configuration update must be an object")
	}
	merge(current, patch)
	return json.Marshal(current)
}

func merge(current, patch map[string]json.RawMessage) {
	for key, raw := range patch {
		raw = bytes.TrimSpace(raw)
		if len(raw) > 0 && raw[0] == '{' {
			var nested, old map[string]json.RawMessage
			_ = json.Unmarshal(raw, &nested)
			_ = json.Unmarshal(current[key], &old)
			if old == nil {
				old = make(map[string]json.RawMessage)
			}
			merge(old, nested)
			current[key], _ = json.Marshal(old)
		} else {
			current[key] = raw
		}
	}
}
