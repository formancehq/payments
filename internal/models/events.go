package models

import "time"

type EventSent struct {
	// Unique Event ID generated from event information
	ID EventID
	// Related Connector ID
	ConnectorID *ConnectorID
	// Time when the event was sent
	SentAt time.Time
}
