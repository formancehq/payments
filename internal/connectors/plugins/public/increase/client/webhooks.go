package client

import (
	"time"
)

type WebhookEvent struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	CreatedAt   time.Time       `json:"created_at"`
	Category    string          `json:"category"`
	Data        map[string]any  `json:"data"`
}
