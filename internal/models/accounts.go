package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/utils/assets"
)

// Internal struct used by the plugins
type PSPAccount struct {
	// PSP reference of the account. Should be unique.
	Reference string

	// Account's creation date
	CreatedAt time.Time

	// Optional, human readable name of the account (if existing)
	Name *string
	// Optional, if provided the default asset of the account
	// in minor currencies unit.
	DefaultAsset *string

	// Additional metadata
	Metadata map[string]string

	// PSP response in raw
	Raw json.RawMessage
}

func (p *PSPAccount) Validate() error {
	if p.Reference == "" {
		return fmt.Errorf("missing account reference: %w", ErrValidation)
	}

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("missing account createdAt: %w", ErrValidation)
	}

	if p.Raw == nil {
		return fmt.Errorf("missing account raw: %w", ErrValidation)
	}

	if p.DefaultAsset != nil && !assets.IsValid(*p.DefaultAsset) {
		return fmt.Errorf("invalid default asset: %w", ErrValidation)
	}

	return nil
}

type Account struct {
	// Unique Account ID generated from account information
	ID AccountID `json:"id"`
	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`

	// PSP reference of the account. Should be unique.
	Reference string `json:"reference"`

	// Account's creation date
	CreatedAt time.Time `json:"createdAt"`

	// Type of account: INTERNAL, EXTERNAL...
	Type AccountType `json:"type"`

	// Optional, human readable name of the account (if existing)
	Name *string `json:"name"`
	// Optional, if provided the default asset of the account
	// in minor currencies unit.
	DefaultAsset *string `json:"defaultAsset"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`

	// PSP response in raw
	Raw json.RawMessage `json:"raw"`
}

func (a Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID           string            `json:"id"`
		ConnectorID  string            `json:"connectorID"`
		Provider     string            `json:"provider"`
		Reference    string            `json:"reference"`
		CreatedAt    time.Time         `json:"createdAt"`
		Type         AccountType       `json:"type"`
		Name         *string           `json:"name"`
		DefaultAsset *string           `json:"defaultAsset"`
		Metadata     map[string]string `json:"metadata"`
		Raw          json.RawMessage   `json:"raw"`
	}{
		ID:           a.ID.String(),
		ConnectorID:  a.ConnectorID.String(),
		Provider:     ToV3Provider(a.ConnectorID.Provider),
		Reference:    a.Reference,
		CreatedAt:    a.CreatedAt,
		Type:         a.Type,
		Name:         a.Name,
		DefaultAsset: a.DefaultAsset,
		Metadata:     a.Metadata,
		Raw:          a.Raw,
	})
}

func (a *Account) IdempotencyKey() string {
	return IdempotencyKey(a.ID)
}

func (a *Account) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID           string            `json:"id"`
		ConnectorID  string            `json:"connectorID"`
		Reference    string            `json:"reference"`
		CreatedAt    time.Time         `json:"createdAt"`
		Type         AccountType       `json:"type"`
		Name         *string           `json:"name"`
		DefaultAsset *string           `json:"defaultAsset"`
		Metadata     map[string]string `json:"metadata"`
		Raw          json.RawMessage   `json:"raw"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := AccountIDFromString(aux.ID)
	if err != nil {
		return err
	}

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}

	a.ID = id
	a.ConnectorID = connectorID
	a.Reference = aux.Reference
	a.CreatedAt = aux.CreatedAt
	a.Type = aux.Type
	a.Name = aux.Name
	a.DefaultAsset = aux.DefaultAsset
	a.Metadata = aux.Metadata
	a.Raw = aux.Raw

	return nil
}

func FromPSPAccount(from PSPAccount, accountType AccountType, connectorID ConnectorID) (Account, error) {
	if err := from.Validate(); err != nil {
		return Account{}, err
	}

	return Account{
		ID: AccountID{
			Reference:   from.Reference,
			ConnectorID: connectorID,
		},
		ConnectorID:  connectorID,
		Reference:    from.Reference,
		CreatedAt:    from.CreatedAt,
		Type:         accountType,
		Name:         from.Name,
		DefaultAsset: from.DefaultAsset,
		Metadata:     from.Metadata,
		Raw:          from.Raw,
	}, nil
}

func FromPSPAccounts(from []PSPAccount, accountType AccountType, connectorID ConnectorID, additionalMetadata map[string]string) ([]Account, error) {
	accounts := make([]Account, 0, len(from))
	for _, a := range from {
		account, err := FromPSPAccount(a, accountType, connectorID)
		if err != nil {
			return nil, err
		}

		for k, v := range additionalMetadata {
			account.Metadata[k] = v
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

func ToPSPAccount(from *Account) *PSPAccount {
	if from == nil {
		return nil
	}
	return &PSPAccount{
		Reference:    from.Reference,
		CreatedAt:    from.CreatedAt,
		Name:         from.Name,
		DefaultAsset: from.DefaultAsset,
		Metadata:     from.Metadata,
		Raw:          from.Raw,
	}
}
