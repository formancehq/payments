package events

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type ConversionMessagePayload struct {
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
	SourceAccountID      string            `json:"sourceAccountID,omitempty"`
	DestinationAccountID string            `json:"destinationAccountID,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	Raw                  json.RawMessage   `json:"raw"`
}

func (p *ConversionMessagePayload) MarshalJSON() ([]byte, error) {
	type Alias ConversionMessagePayload
	return json.Marshal(&struct {
		SourceAmount      *string `json:"sourceAmount"`
		DestinationAmount *string `json:"destinationAmount,omitempty"`
		Fee               *string `json:"fee,omitempty"`
		*Alias
	}{
		SourceAmount:      bigIntToString(p.SourceAmount),
		DestinationAmount: bigIntToString(p.DestinationAmount),
		Fee:               bigIntToString(p.Fee),
		Alias:             (*Alias)(p),
	})
}

func (p *ConversionMessagePayload) UnmarshalJSON(data []byte) error {
	type Alias ConversionMessagePayload
	aux := &struct {
		SourceAmount      *string `json:"sourceAmount"`
		DestinationAmount *string `json:"destinationAmount,omitempty"`
		Fee               *string `json:"fee,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if p.SourceAmount, err = bigIntFromString(aux.SourceAmount, "sourceAmount"); err != nil {
		return err
	}
	if p.DestinationAmount, err = bigIntFromString(aux.DestinationAmount, "destinationAmount"); err != nil {
		return err
	}
	if p.Fee, err = bigIntFromString(aux.Fee, "fee"); err != nil {
		return err
	}
	return nil
}

func (e Events) NewEventSavedConversion(conversion models.Conversion) publish.EventMessage {
	payload := ConversionMessagePayload{
		ID:               conversion.ID.String(),
		ConnectorID:      conversion.ConnectorID.String(),
		Provider:         models.ToV3Provider(conversion.ConnectorID.Provider),
		Reference:        conversion.Reference,
		CreatedAt:        conversion.CreatedAt,
		UpdatedAt:        conversion.UpdatedAt,
		SourceAsset:      conversion.SourceAsset,
		DestinationAsset: conversion.DestinationAsset,
		SourceAmount:     conversion.SourceAmount,
		DestinationAmount: conversion.DestinationAmount,
		Fee:              conversion.Fee,
		FeeAsset:         conversion.FeeAsset,
		Status:           conversion.Status.String(),
		SourceAccountID: func() string {
			if conversion.SourceAccountID == nil {
				return ""
			}
			return conversion.SourceAccountID.String()
		}(),
		DestinationAccountID: func() string {
			if conversion.DestinationAccountID == nil {
				return ""
			}
			return conversion.DestinationAccountID.String()
		}(),
		Metadata: conversion.Metadata,
		Raw:      conversion.Raw,
	}

	return publish.EventMessage{
		IdempotencyKey: conversion.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedConversion,
		Payload:        payload,
	}
}
