package payment

type Connector struct {
	Provider string      `json:"provider" bson:"provider"`
	Disabled bool        `json:"disabled" bson:"disabled"`
	Config   interface{} `json:"config" bson:"config"`
	State    interface{} `json:"state" bson:"state"`
}
