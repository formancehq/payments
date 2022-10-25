package payments

type ConnectorBaseInfo struct {
	Provider string `json:"provider" bson:"provider"`
	Disabled bool   `json:"disabled" bson:"disabled"`
}

type Connector[T ConnectorConfigObject] struct {
	ConnectorBaseInfo
	Config T `json:"config" bson:"config"`
}
