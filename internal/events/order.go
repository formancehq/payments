package events

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type OrderAdjustmentPayload struct {
	ID                 string            `json:"id"`
	Reference          string            `json:"reference"`
	CreatedAt          time.Time         `json:"createdAt"`
	Status             string            `json:"status"`
	BaseQuantityFilled *big.Int          `json:"baseQuantityFilled,omitempty"`
	Fee                *big.Int          `json:"fee,omitempty"`
	FeeAsset           *string           `json:"feeAsset,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	Raw                json.RawMessage   `json:"raw"`
}

type OrderMessagePayload struct {
	ID                   string                   `json:"id"`
	ConnectorID          string                   `json:"connectorID"`
	Provider             string                   `json:"provider"`
	Reference            string                   `json:"reference"`
	ClientOrderID        string                   `json:"clientOrderId,omitempty"`
	CreatedAt            time.Time                `json:"createdAt"`
	UpdatedAt            time.Time                `json:"updatedAt"`
	Direction            string                   `json:"direction"`
	SourceAsset          string                   `json:"sourceAsset"`
	DestinationAsset     string                   `json:"destinationAsset"`
	Type                 string                   `json:"type"`
	Status               string                   `json:"status"`
	TimeInForce          string                   `json:"timeInForce"`
	ExpiresAt            *time.Time               `json:"expiresAt,omitempty"`
	BaseQuantityOrdered  *big.Int                 `json:"baseQuantityOrdered"`
	BaseQuantityFilled   *big.Int                 `json:"baseQuantityFilled,omitempty"`
	LimitPrice           *big.Int                 `json:"limitPrice,omitempty"`
	StopPrice            *big.Int                 `json:"stopPrice,omitempty"`
	QuoteAmount          *big.Int                 `json:"quoteAmount,omitempty"`
	QuoteAsset           string                   `json:"quoteAsset,omitempty"`
	Fee                  *big.Int                 `json:"fee,omitempty"`
	FeeAsset             *string                  `json:"feeAsset,omitempty"`
	AverageFillPrice     *big.Int                 `json:"averageFillPrice,omitempty"`
	PriceAsset           *string                  `json:"priceAsset,omitempty"`
	SourceAccountID      string                   `json:"sourceAccountID,omitempty"`
	DestinationAccountID string                   `json:"destinationAccountID,omitempty"`
	RawData              json.RawMessage           `json:"rawData"`
	Metadata             map[string]string         `json:"metadata,omitempty"`
	Adjustments          []OrderAdjustmentPayload  `json:"adjustments,omitempty"`
}

func (p *OrderMessagePayload) MarshalJSON() ([]byte, error) {
	type Alias OrderMessagePayload
	return json.Marshal(&struct {
		BaseQuantityOrdered *string `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *string `json:"baseQuantityFilled,omitempty"`
		LimitPrice          *string `json:"limitPrice,omitempty"`
		StopPrice           *string `json:"stopPrice,omitempty"`
		QuoteAmount         *string `json:"quoteAmount,omitempty"`
		Fee                 *string `json:"fee,omitempty"`
		AverageFillPrice    *string `json:"averageFillPrice,omitempty"`
		*Alias
	}{
		BaseQuantityOrdered: bigIntToString(p.BaseQuantityOrdered),
		BaseQuantityFilled:  bigIntToString(p.BaseQuantityFilled),
		LimitPrice:          bigIntToString(p.LimitPrice),
		StopPrice:           bigIntToString(p.StopPrice),
		QuoteAmount:         bigIntToString(p.QuoteAmount),
		Fee:                 bigIntToString(p.Fee),
		AverageFillPrice:    bigIntToString(p.AverageFillPrice),
		Alias:               (*Alias)(p),
	})
}

func (p *OrderMessagePayload) UnmarshalJSON(data []byte) error {
	type Alias OrderMessagePayload
	aux := &struct {
		BaseQuantityOrdered *string `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *string `json:"baseQuantityFilled,omitempty"`
		LimitPrice          *string `json:"limitPrice,omitempty"`
		StopPrice           *string `json:"stopPrice,omitempty"`
		QuoteAmount         *string `json:"quoteAmount,omitempty"`
		Fee                 *string `json:"fee,omitempty"`
		AverageFillPrice    *string `json:"averageFillPrice,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if p.BaseQuantityOrdered, err = bigIntFromString(aux.BaseQuantityOrdered, "baseQuantityOrdered"); err != nil {
		return err
	}
	if p.BaseQuantityFilled, err = bigIntFromString(aux.BaseQuantityFilled, "baseQuantityFilled"); err != nil {
		return err
	}
	if p.LimitPrice, err = bigIntFromString(aux.LimitPrice, "limitPrice"); err != nil {
		return err
	}
	if p.StopPrice, err = bigIntFromString(aux.StopPrice, "stopPrice"); err != nil {
		return err
	}
	if p.QuoteAmount, err = bigIntFromString(aux.QuoteAmount, "quoteAmount"); err != nil {
		return err
	}
	if p.Fee, err = bigIntFromString(aux.Fee, "fee"); err != nil {
		return err
	}
	if p.AverageFillPrice, err = bigIntFromString(aux.AverageFillPrice, "averageFillPrice"); err != nil {
		return err
	}
	return nil
}

func (e Events) NewEventSavedOrder(order models.Order, adjustment models.OrderAdjustment) publish.EventMessage {
	adjustments := make([]OrderAdjustmentPayload, 0, len(order.Adjustments))
	for _, a := range order.Adjustments {
		adjustments = append(adjustments, OrderAdjustmentPayload{
			ID:                 a.ID.String(),
			Reference:          a.Reference,
			CreatedAt:          a.CreatedAt,
			Status:             a.Status.String(),
			BaseQuantityFilled: a.BaseQuantityFilled,
			Fee:                a.Fee,
			FeeAsset:           a.FeeAsset,
			Metadata:           a.Metadata,
			Raw:                a.Raw,
		})
	}

	payload := OrderMessagePayload{
		ID:                  order.ID.String(),
		ConnectorID:         order.ConnectorID.String(),
		Provider:            models.ToV3Provider(order.ConnectorID.Provider),
		Reference:           order.Reference,
		ClientOrderID:       order.ClientOrderID,
		CreatedAt:           order.CreatedAt,
		UpdatedAt:           order.UpdatedAt,
		Direction:           order.Direction.String(),
		SourceAsset:         order.SourceAsset,
		DestinationAsset:    order.DestinationAsset,
		Type:                order.Type.String(),
		Status:              order.Status.String(),
		TimeInForce:         order.TimeInForce.String(),
		ExpiresAt:           order.ExpiresAt,
		BaseQuantityOrdered: order.BaseQuantityOrdered,
		BaseQuantityFilled:  order.BaseQuantityFilled,
		LimitPrice:          order.LimitPrice,
		StopPrice:           order.StopPrice,
		QuoteAmount:         order.QuoteAmount,
		QuoteAsset:          order.QuoteAsset,
		Fee:                 order.Fee,
		FeeAsset:            order.FeeAsset,
		AverageFillPrice:    order.AverageFillPrice,
		PriceAsset:          order.PriceAsset,
		SourceAccountID: func() string {
			if order.SourceAccountID == nil {
				return ""
			}
			return order.SourceAccountID.String()
		}(),
		DestinationAccountID: func() string {
			if order.DestinationAccountID == nil {
				return ""
			}
			return order.DestinationAccountID.String()
		}(),
		RawData:     adjustment.Raw,
		Metadata:    order.Metadata,
		Adjustments: adjustments,
	}

	return publish.EventMessage{
		IdempotencyKey: adjustment.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedOrder,
		Payload:        payload,
	}
}
