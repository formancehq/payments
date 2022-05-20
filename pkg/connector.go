package payments

type Connector[T ConnectorConfigObject, S ConnectorState] struct {
	Provider string `json:"provider" bson:"provider"`
	Disabled bool   `json:"disabled" bson:"disabled"`
	Config   T      `json:"config" bson:"config"`
	State    S      `json:"state" bson:"state"`
}
