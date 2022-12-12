package payments

import "github.com/formancehq/payments/internal/app/models"

type ConnectorBaseInfo struct {
	Provider models.ConnectorProvider `json:"provider" bson:"provider"`
	Disabled bool                     `json:"disabled" bson:"disabled"`
}

type Connector[T ConnectorConfigObject] struct {
	ConnectorBaseInfo
	Config T `json:"config" bson:"config"`
}
