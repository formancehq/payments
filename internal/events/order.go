package events

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type OrderMessagePayload struct {
	ID                  string            `json:"id"`
	ConnectorID         string            `json:"connectorID"`
	Provider            string            `json:"provider"`
	Reference           string            `json:"reference"`
	ClientOrderID       string            `json:"clientOrderId,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	Direction           string            `json:"direction"`
	SourceAsset         string            `json:"sourceAsset"`
	DestinationAsset         string            `json:"destinationAsset"`
	Type                string            `json:"type"`
	Status              string            `json:"status"`
	BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered"`
	BaseQuantityFilled  *big.Int          `json:"baseQuantityFilled,omitempty"`
	Fee                 *big.Int          `json:"fee,omitempty"`
	FeeAsset            *string           `json:"feeAsset,omitempty"`
	RawData             json.RawMessage   `json:"rawData"`
	Metadata            map[string]string `json:"metadata,omitempty"`
}

func (p *OrderMessagePayload) MarshalJSON() ([]byte, error) {
	type Alias OrderMessagePayload
	return json.Marshal(&struct {
		BaseQuantityOrdered *string `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *string `json:"baseQuantityFilled,omitempty"`
		Fee                 *string `json:"fee,omitempty"`
		*Alias
	}{
		BaseQuantityOrdered: bigIntToString(p.BaseQuantityOrdered),
		BaseQuantityFilled:  bigIntToString(p.BaseQuantityFilled),
		Fee:                 bigIntToString(p.Fee),
		Alias:               (*Alias)(p),
	})
}

func (p *OrderMessagePayload) UnmarshalJSON(data []byte) error {
	type Alias OrderMessagePayload
	aux := &struct {
		BaseQuantityOrdered *string `json:"baseQuantityOrdered"`
		BaseQuantityFilled  *string `json:"baseQuantityFilled,omitempty"`
		Fee                 *string `json:"fee,omitempty"`
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
	if p.Fee, err = bigIntFromString(aux.Fee, "fee"); err != nil {
		return err
	}
	return nil
}

func (e Events) NewEventSavedOrder(order models.Order, adjustment models.OrderAdjustment) publish.EventMessage {
	payload := OrderMessagePayload{
		ID:                  order.ID.String(),
		ConnectorID:         order.ConnectorID.String(),
		Provider:            models.ToV3Provider(order.ConnectorID.Provider),
		Reference:           order.Reference,
		ClientOrderID:       order.ClientOrderID,
		CreatedAt:           order.CreatedAt,
		Direction:           order.Direction.String(),
		SourceAsset:         order.SourceAsset,
		DestinationAsset:         order.DestinationAsset,
		Type:                order.Type.String(),
		Status:              order.Status.String(),
		BaseQuantityOrdered: order.BaseQuantityOrdered,
		BaseQuantityFilled:  order.BaseQuantityFilled,
		Fee:                 order.Fee,
		FeeAsset:            order.FeeAsset,
		RawData:             adjustment.Raw,
		Metadata:            order.Metadata,
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
