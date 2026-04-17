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

// PSPConversion represents an asset conversion (stablecoin, FX, wrapped asset swap).
type PSPConversion struct {
	Reference string
	CreatedAt time.Time

	SourceAsset       string
	DestinationAsset  string
	SourceAmount      *big.Int
	DestinationAmount *big.Int

	Fee      *big.Int
	FeeAsset *string

	Status ConversionStatus

	SourceAccountReference      *string
	DestinationAccountReference *string

	Metadata map[string]string
	Raw      json.RawMessage
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
	if !assets.IsValid(c.DestinationAsset) {
		return errorsutils.NewWrappedError(errors.New("invalid conversion destination asset"), ErrValidation)
	}
	if c.SourceAmount == nil {
		return errorsutils.NewWrappedError(errors.New("missing conversion source amount"), ErrValidation)
	}
	if c.Status == CONVERSION_STATUS_UNKNOWN {
		return errorsutils.NewWrappedError(errors.New("missing conversion status"), ErrValidation)
	}
	if c.Raw == nil {
		return errorsutils.NewWrappedError(errors.New("missing conversion raw"), ErrValidation)
	}
	return nil
}

// Conversion represents an asset conversion in Formance.
type Conversion struct {
	ID                ConversionID     `json:"id"`
	ConnectorID       ConnectorID      `json:"connectorID"`
	Reference         string           `json:"reference"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
	SourceAsset       string           `json:"sourceAsset"`
	DestinationAsset  string           `json:"destinationAsset"`
	SourceAmount      *big.Int         `json:"sourceAmount"`
	DestinationAmount *big.Int         `json:"destinationAmount,omitempty"`
	Fee               *big.Int         `json:"fee,omitempty"`
	FeeAsset          *string          `json:"feeAsset,omitempty"`
	Status            ConversionStatus `json:"status"`

	SourceAccountID      *AccountID `json:"sourceAccountID"`
	DestinationAccountID *AccountID `json:"destinationAccountID"`

	Metadata map[string]string `json:"metadata"`
	Raw      json.RawMessage   `json:"raw"`
}

func (c *Conversion) IdempotencyKey() string {
	return IdempotencyKey(struct {
		ID     ConversionID     `json:"ID"`
		Status ConversionStatus `json:"Status"`
	}{c.ID, c.Status})
}

func (c Conversion) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                   string            `json:"id"`
		ConnectorID          string            `json:"connectorID"`
		Provider             string            `json:"provider"`
		Reference            string            `json:"reference"`
		CreatedAt            time.Time         `json:"createdAt"`
		UpdatedAt            time.Time         `json:"updatedAt"`
		SourceAsset          string            `json:"sourceAsset"`
		DestinationAsset     string            `json:"destinationAsset"`
		SourceAmount         *big.Int          `json:"sourceAmount"`
		DestinationAmount    *big.Int          `json:"destinationAmount,omitempty"`
		Fee                  *big.Int          `json:"fee,omitempty"`
		FeeAsset             *string           `json:"feeAsset,omitempty"`
		Status               ConversionStatus  `json:"status"`
		SourceAccountID      *string           `json:"sourceAccountID"`
		DestinationAccountID *string           `json:"destinationAccountID"`
		Metadata             map[string]string `json:"metadata"`
		Raw                  json.RawMessage   `json:"raw"`
	}{
		ID:               c.ID.String(),
		ConnectorID:      c.ConnectorID.String(),
		Provider:         ToV3Provider(c.ConnectorID.Provider),
		Reference:        c.Reference,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
		SourceAsset:      c.SourceAsset,
		DestinationAsset: c.DestinationAsset,
		SourceAmount:     c.SourceAmount,
		DestinationAmount: c.DestinationAmount,
		Fee:              c.Fee,
		FeeAsset:         c.FeeAsset,
		Status:           c.Status,
		SourceAccountID:      c.SourceAccountID.StringPtr(),
		DestinationAccountID: c.DestinationAccountID.StringPtr(),
		Metadata: c.Metadata,
		Raw:      c.Raw,
	})
}

func (c *Conversion) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID                   string            `json:"id"`
		ConnectorID          string            `json:"connectorID"`
		Reference            string            `json:"reference"`
		CreatedAt            time.Time         `json:"createdAt"`
		UpdatedAt            time.Time         `json:"updatedAt"`
		SourceAsset          string            `json:"sourceAsset"`
		DestinationAsset     string            `json:"destinationAsset"`
		SourceAmount         *big.Int          `json:"sourceAmount"`
		DestinationAmount    *big.Int          `json:"destinationAmount,omitempty"`
		Fee                  *big.Int          `json:"fee,omitempty"`
		FeeAsset             *string           `json:"feeAsset,omitempty"`
		Status               ConversionStatus  `json:"status"`
		SourceAccountID      *string           `json:"sourceAccountID"`
		DestinationAccountID *string           `json:"destinationAccountID"`
		Metadata             map[string]string `json:"metadata"`
		Raw                  json.RawMessage   `json:"raw"`
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
	c.DestinationAsset = aux.DestinationAsset
	c.SourceAmount = aux.SourceAmount
	c.DestinationAmount = aux.DestinationAmount
	c.Fee = aux.Fee
	c.FeeAsset = aux.FeeAsset
	c.Status = aux.Status
	if aux.SourceAccountID != nil {
		id, err := AccountIDFromString(*aux.SourceAccountID)
		if err != nil {
			return err
		}
		c.SourceAccountID = &id
	}
	if aux.DestinationAccountID != nil {
		id, err := AccountIDFromString(*aux.DestinationAccountID)
		if err != nil {
			return err
		}
		c.DestinationAccountID = &id
	}
	c.Metadata = aux.Metadata
	c.Raw = aux.Raw

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
		ConnectorID:          connectorID,
		Reference:            from.Reference,
		CreatedAt:            from.CreatedAt,
		UpdatedAt:            now,
		SourceAsset:          from.SourceAsset,
		DestinationAsset:     from.DestinationAsset,
		SourceAmount:         from.SourceAmount,
		DestinationAmount:    from.DestinationAmount,
		Fee:                  from.Fee,
		FeeAsset:             from.FeeAsset,
		Status:               from.Status,
		SourceAccountID:      NewAccountID(from.SourceAccountReference, connectorID),
		DestinationAccountID: NewAccountID(from.DestinationAccountReference, connectorID),
		Metadata:             from.Metadata,
		Raw:                  from.Raw,
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
		ID                   string            `json:"id"`
		ConnectorID          string            `json:"connectorID"`
		Provider             string            `json:"provider"`
		Reference            string            `json:"reference"`
		CreatedAt            time.Time         `json:"createdAt"`
		UpdatedAt            time.Time         `json:"updatedAt"`
		SourceAsset          string            `json:"sourceAsset"`
		DestinationAsset     string            `json:"destinationAsset"`
		SourceAmount         *big.Int          `json:"sourceAmount"`
		DestinationAmount    *big.Int          `json:"destinationAmount,omitempty"`
		Fee                  *big.Int          `json:"fee,omitempty"`
		FeeAsset             *string           `json:"feeAsset,omitempty"`
		Status               string            `json:"status"`
		SourceAccountID      *string           `json:"sourceAccountID"`
		DestinationAccountID *string           `json:"destinationAccountID"`
		Metadata             map[string]string `json:"metadata"`
		Raw                  json.RawMessage   `json:"raw"`
		Error                *string           `json:"error,omitempty"`
	}{
		ID:                   ce.Conversion.ID.String(),
		ConnectorID:          ce.Conversion.ConnectorID.String(),
		Provider:             ToV3Provider(ce.Conversion.ConnectorID.Provider),
		Reference:            ce.Conversion.Reference,
		CreatedAt:            ce.Conversion.CreatedAt,
		UpdatedAt:            ce.Conversion.UpdatedAt,
		SourceAsset:          ce.Conversion.SourceAsset,
		DestinationAsset:     ce.Conversion.DestinationAsset,
		SourceAmount:         ce.Conversion.SourceAmount,
		DestinationAmount:    ce.Conversion.DestinationAmount,
		Fee:                  ce.Conversion.Fee,
		FeeAsset:             ce.Conversion.FeeAsset,
		Status:               ce.Status.String(),
		SourceAccountID:      ce.Conversion.SourceAccountID.StringPtr(),
		DestinationAccountID: ce.Conversion.DestinationAccountID.StringPtr(),
		Metadata:             ce.Conversion.Metadata,
		Raw:                  ce.Conversion.Raw,
		Error: func() *string {
			if ce.Error == nil {
				return nil
			}
			return pointer.For(ce.Error.Error())
		}(),
	})
}
