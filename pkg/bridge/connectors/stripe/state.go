package stripe

import "time"

type TimelineState struct {
	OldestID       string     `bson:"oldestID,omitempty" json:"oldestID"`
	OldestDate     *time.Time `bson:"oldestDate,omitempty" json:"oldestDate"`
	MoreRecentID   string     `bson:"moreRecentID,omitempty" json:"moreRecentID"`
	MoreRecentDate *time.Time `bson:"moreRecentDate,omitempty" json:"moreRecentDate"`
}

type State struct {
	TimelineState `bson:",inline"`
	Accounts      map[string]TimelineState `bson:"accounts,omitempty"`
}
