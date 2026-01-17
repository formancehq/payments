package models

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/utils/assets"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

// PSPConversion represents a stablecoin conversion from a Payment Service Provider.
// Used for USD↔USDC and USD↔PYUSD conversions on Coinbase Prime.
type PSPConversion struct {
	// PSP conversion reference. Should be unique within the connector.
	Reference string

	// Conversion creation date
	CreatedAt time.Time

	// Source asset (what you're converting from, e.g., USD/2)
	SourceAsset string

	// Target asset (what you're converting to, e.g., USDC/6)
	TargetAsset string

	// Source amount (using integer representation)
	SourceAmount *big.Int

	// Target amount (using integer representation, may be nil for pending conversions)
	TargetAmount *big.Int

	// Conversion status
	Status ConversionStatus

	// Wallet ID where the conversion takes place (required by Coinbase Prime)
	WalletID string

	// Additional metadata
	Metadata map[string]string

	// PSP response in raw format
	Raw json.RawMessage
}

func (c *PSPConversion) Validate() error {
	if c.Reference == "" {
		return errorsutils.NewWrappedError(errors.New("missing conversion reference"), ErrValidation)
	}

	if c.CreatedAt.IsZero() {
		return errorsutils.NewWrappedError(errors.New("missing conversion createdAt"), ErrValidation)
	}

	if !assets.IsValid(c.SourceAsset) {
		return errorsutils.NewWrappedError(errors.New("invalid conversion source asset"), ErrValidation)
	}

	if !assets.IsValid(c.TargetAsset) {
		return errorsutils.NewWrappedError(errors.New("invalid conversion target asset"), ErrValidation)
	}

	if c.SourceAmount == nil {
		return errorsutils.NewWrappedError(errors.New("missing conversion source amount"), ErrValidation)
	}

	if c.Status == CONVERSION_STATUS_UNKNOWN {
		return errorsutils.NewWrappedError(errors.New("missing conversion status"), ErrValidation)
	}

	if c.WalletID == "" {
		return errorsutils.NewWrappedError(errors.New("missing conversion wallet id"), ErrValidation)
	}

	if c.Raw == nil {
		return errorsutils.NewWrappedError(errors.New("missing conversion raw"), ErrValidation)
	}

	return nil
}

// Conversion represents a stablecoin conversion in Formance.
type Conversion struct {
	// Unique Conversion ID generated from conversion information
	ID ConversionID `json:"id"`

	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`

	// PSP conversion reference
	Reference string `json:"reference"`

	// Conversion creation date
	CreatedAt time.Time `json:"createdAt"`

	// Last update date
	UpdatedAt time.Time `json:"updatedAt"`

	// Source asset
	SourceAsset string `json:"sourceAsset"`

	// Target asset
	TargetAsset string `json:"targetAsset"`

	// Source amount (using integer representation)
	SourceAmount *big.Int `json:"sourceAmount"`

	// Target amount (using integer representation)
	TargetAmount *big.Int `json:"targetAmount,omitempty"`

	// Conversion status
	Status ConversionStatus `json:"status"`

	// Wallet ID where the conversion takes place
	WalletID string `json:"walletId"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`
}

func (c *Conversion) IdempotencyKey() string {
	return IdempotencyKey(c.ID)
}

func (c Conversion) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID           string            `json:"id"`
		ConnectorID  string            `json:"connectorID"`
		Provider     string            `json:"provider"`
		Reference    string            `json:"reference"`
		CreatedAt    time.Time         `json:"createdAt"`
		UpdatedAt    time.Time         `json:"updatedAt"`
		SourceAsset  string            `json:"sourceAsset"`
		TargetAsset  string            `json:"targetAsset"`
		SourceAmount *big.Int          `json:"sourceAmount"`
		TargetAmount *big.Int          `json:"targetAmount,omitempty"`
		Status       ConversionStatus  `json:"status"`
		WalletID     string            `json:"walletId"`
		Metadata     map[string]string `json:"metadata"`
	}{
		ID:           c.ID.String(),
		ConnectorID:  c.ConnectorID.String(),
		Provider:     ToV3Provider(c.ConnectorID.Provider),
		Reference:    c.Reference,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
		SourceAsset:  c.SourceAsset,
		TargetAsset:  c.TargetAsset,
		SourceAmount: c.SourceAmount,
		TargetAmount: c.TargetAmount,
		Status:       c.Status,
		WalletID:     c.WalletID,
		Metadata:     c.Metadata,
	})
}

func (c *Conversion) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID           string            `json:"id"`
		ConnectorID  string            `json:"connectorID"`
		Reference    string            `json:"reference"`
		CreatedAt    time.Time         `json:"createdAt"`
		UpdatedAt    time.Time         `json:"updatedAt"`
		SourceAsset  string            `json:"sourceAsset"`
		TargetAsset  string            `json:"targetAsset"`
		SourceAmount *big.Int          `json:"sourceAmount"`
		TargetAmount *big.Int          `json:"targetAmount,omitempty"`
		Status       ConversionStatus  `json:"status"`
		WalletID     string            `json:"walletId"`
		Metadata     map[string]string `json:"metadata"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	id, err := ConversionIDFromString(aux.ID)
	if err != nil {
		return err
	}

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}

	c.ID = id
	c.ConnectorID = connectorID
	c.Reference = aux.Reference
	c.CreatedAt = aux.CreatedAt
	c.UpdatedAt = aux.UpdatedAt
	c.SourceAsset = aux.SourceAsset
	c.TargetAsset = aux.TargetAsset
	c.SourceAmount = aux.SourceAmount
	c.TargetAmount = aux.TargetAmount
	c.Status = aux.Status
	c.WalletID = aux.WalletID
	c.Metadata = aux.Metadata

	return nil
}

// FromPSPConversionToConversion converts a PSPConversion to a Conversion
func FromPSPConversionToConversion(from PSPConversion, connectorID ConnectorID) (Conversion, error) {
	if err := from.Validate(); err != nil {
		return Conversion{}, err
	}

	now := time.Now().UTC()
	return Conversion{
		ID: ConversionID{
			Reference:   from.Reference,
			ConnectorID: connectorID,
		},
		ConnectorID:  connectorID,
		Reference:    from.Reference,
		CreatedAt:    from.CreatedAt,
		UpdatedAt:    now,
		SourceAsset:  from.SourceAsset,
		TargetAsset:  from.TargetAsset,
		SourceAmount: from.SourceAmount,
		TargetAmount: from.TargetAmount,
		Status:       from.Status,
		WalletID:     from.WalletID,
		Metadata:     from.Metadata,
	}, nil
}

// FromPSPConversions converts a slice of PSPConversions to Conversions
func FromPSPConversions(from []PSPConversion, connectorID ConnectorID) ([]Conversion, error) {
	conversions := make([]Conversion, 0, len(from))
	for _, c := range from {
		conversion, err := FromPSPConversionToConversion(c, connectorID)
		if err != nil {
			return nil, err
		}
		conversions = append(conversions, conversion)
	}
	return conversions, nil
}

// ConversionExpanded includes conversion with its current status and optional error
type ConversionExpanded struct {
	Conversion Conversion
	Status     ConversionStatus
	Error      error
}

func (ce ConversionExpanded) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID           string            `json:"id"`
		ConnectorID  string            `json:"connectorID"`
		Provider     string            `json:"provider"`
		Reference    string            `json:"reference"`
		CreatedAt    time.Time         `json:"createdAt"`
		UpdatedAt    time.Time         `json:"updatedAt"`
		SourceAsset  string            `json:"sourceAsset"`
		TargetAsset  string            `json:"targetAsset"`
		SourceAmount *big.Int          `json:"sourceAmount"`
		TargetAmount *big.Int          `json:"targetAmount,omitempty"`
		Status       string            `json:"status"`
		WalletID     string            `json:"walletId"`
		Metadata     map[string]string `json:"metadata"`
		Error        *string           `json:"error,omitempty"`
	}{
		ID:           ce.Conversion.ID.String(),
		ConnectorID:  ce.Conversion.ConnectorID.String(),
		Provider:     ToV3Provider(ce.Conversion.ConnectorID.Provider),
		Reference:    ce.Conversion.Reference,
		CreatedAt:    ce.Conversion.CreatedAt,
		UpdatedAt:    ce.Conversion.UpdatedAt,
		SourceAsset:  ce.Conversion.SourceAsset,
		TargetAsset:  ce.Conversion.TargetAsset,
		SourceAmount: ce.Conversion.SourceAmount,
		TargetAmount: ce.Conversion.TargetAmount,
		Status:       ce.Status.String(),
		WalletID:     ce.Conversion.WalletID,
		Metadata:     ce.Conversion.Metadata,
		Error: func() *string {
			if ce.Error == nil {
				return nil
			}
			return pointer.For(ce.Error.Error())
		}(),
	})
}
