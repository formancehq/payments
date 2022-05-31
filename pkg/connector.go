package payments

type Connector[T ConnectorConfigObject] struct {
	Provider string `json:"provider" bson:"provider"`
	Disabled bool   `json:"disabled" bson:"disabled"`
	Config   T      `json:"config" bson:"config"`
}

type State[S ConnectorState] struct {
	Provider string `json:"provider" bson:"provider"`
	State    S      `json:"state" bson:"state"`
}
