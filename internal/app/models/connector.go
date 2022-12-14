package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"

	"github.com/google/uuid"
)

type Connector struct {
	bun.BaseModel `bun:"connectors.connector"`

	ID        uuid.UUID `bun:",pk,nullzero"`
	CreatedAt time.Time `bun:",nullzero"`
	Provider  ConnectorProvider
	Enabled   bool

	// TODO: Enable DB-level encryption
	Config json.RawMessage

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
	return strings.ToLower(string(p))
}

func (c Connector) ParseConfig(to interface{}) error {
	if c.Config == nil {
		return nil
	}

	err := json.Unmarshal(c.Config, to)
	if err != nil {
		return fmt.Errorf("failed to parse config (%s): %w", string(c.Config), err)
	}

	return nil
}
