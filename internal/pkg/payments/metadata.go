package payments

import (
	"fmt"
	"time"
)

type Metadata map[string]any

type MetadataChanges struct {
	PaymentID  string    `json:"paymentID" bson:"paymentID"`
	OccurredAt time.Time `json:"occurredAt" bson:"occurredAt"`
	Before     Metadata  `json:"before" bson:"before"`
	After      Metadata  `json:"after" bson:"after"`
}

func (mc MetadataChanges) HasChanged() bool {
	if mc.Before == nil {
		return false
	}

	return !mc.Before.Equal(mc.After)
}

func (p *Payment) MergeMetadata(metadata Metadata) MetadataChanges {
	changes := MetadataChanges{
		PaymentID:  p.Identifier.String(),
		OccurredAt: time.Now(),
		Before:     copyMap(p.Metadata),
	}

	if p.Metadata == nil {
		p.Metadata = make(Metadata)
	}

	for key, value := range metadata {
		p.Metadata[key] = value
	}

	changes.After = p.Metadata

	return changes
}

func (m Metadata) Equal(comparable Metadata) bool {
	if len(m) != len(comparable) {
		return false
	}

	for key, value := range m {
		if v, ok := comparable[key]; !ok || fmt.Sprint(v) != fmt.Sprint(value) {
			return false
		}
	}

	return true
}

func copyMap[K string, V any](m map[K]V) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		result[k] = v
	}

	return result
}
