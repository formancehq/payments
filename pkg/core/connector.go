package core

type Connector[T ConnectorConfigObject] struct {
	Provider string `json:"provider" bson:"provider"`
	Disabled bool   `json:"disabled" bson:"disabled"`
	Config   T      `json:"config" bson:"config"`
}
