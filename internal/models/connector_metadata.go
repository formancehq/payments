package models

import "time"

// ConnectorMetadata is a lightweight view of a connector containing only
// the fields required by workflows needing polling/scheduling information.
type ConnectorMetadata struct {
	ConnectorID          ConnectorID
	Provider             string
	PollingPeriod        time.Duration
	ScheduledForDeletion bool
}
