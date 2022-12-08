package models

import (
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Connector struct {
	bun.BaseModel `bun:"connectors.connector"`

	ID        uuid.UUID
	CreatedAt time.Time
	Provider  ConnectorProvider
	Enabled   bool

	// TODO: Enable DB-level encryption
	Config any

	Tasks    []*Task    `bun:"rel:has-many,join:id=connector_id"`
	Payments []*Payment `bun:"rel:has-many,join:id=connector_id"`
}

type ConnectorProvider string

const (
	ConnectorProviderBankingCircle ConnectorProvider = "BANKING-CIRCLE"
	ConnectorProviderCurrencyCloud ConnectorProvider = "CURRENCY-CLOUD"
	ConnectorProviderDummyPay      ConnectorProvider = "DUMMY-PAY"
	ConnectorProviderModulr        ConnectorProvider = "MODULR"
	ConnectorProviderStripe        ConnectorProvider = "STRIPE"
	ConnectorProviderWise          ConnectorProvider = "WISE"
)

func (p ConnectorProvider) String() string {
	return string(p)
}
