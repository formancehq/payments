package events

import (
	"time"

	"github.com/formancehq/go-libs/v3/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
)

type TradeMarketPayload struct {
	Symbol     string `json:"symbol"`
	BaseAsset  string `json:"baseAsset"`
	QuoteAsset string `json:"quoteAsset"`
}

type TradeMessagePayload struct {
	// Mandatory fields
	ID                 string             `json:"id"`
	ConnectorID        string             `json:"connectorID"`
	Provider           string             `json:"provider"`
	Reference          string             `json:"reference"`
	CreatedAt          time.Time          `json:"createdAt"`
	UpdatedAt          time.Time          `json:"updatedAt"`
	InstrumentType     string             `json:"instrumentType"`
	ExecutionModel     string             `json:"executionModel"`
	Market             TradeMarketPayload `json:"market"`
	Side               string             `json:"side"`
	Status             string             `json:"status"`
	Metadata           map[string]string  `json:"metadata,omitempty"`

	// Optional fields
	PortfolioAccountID string `json:"portfolioAccountID,omitempty"`
	OrderType          string `json:"orderType,omitempty"`
	TimeInForce        string `json:"timeInForce,omitempty"`
}

func (e Events) NewEventSavedTrades(trade models.Trade) publish.EventMessage {
	payload := TradeMessagePayload{
		ID:             trade.ID.String(),
		ConnectorID:    trade.ConnectorID.String(),
		Provider:       models.ToV3Provider(trade.ConnectorID.Provider),
		Reference:      trade.Reference,
		CreatedAt:      trade.CreatedAt,
		UpdatedAt:      trade.UpdatedAt,
		InstrumentType: trade.InstrumentType.String(),
		ExecutionModel: trade.ExecutionModel.String(),
		Market: TradeMarketPayload{
			Symbol:     trade.Market.Symbol,
			BaseAsset:  trade.Market.BaseAsset,
			QuoteAsset: trade.Market.QuoteAsset,
		},
		Side:     trade.Side.String(),
		Status:   trade.Status.String(),
		Metadata: trade.Metadata,
	}

	if trade.PortfolioAccountID != nil {
		payload.PortfolioAccountID = trade.PortfolioAccountID.String()
	}

	if trade.OrderType != nil {
		payload.OrderType = trade.OrderType.String()
	}

	if trade.TimeInForce != nil {
		payload.TimeInForce = trade.TimeInForce.String()
	}

	return publish.EventMessage{
		IdempotencyKey: trade.IdempotencyKey(),
		Date:           time.Now().UTC(),
		App:            events.EventApp,
		Version:        events.EventVersion,
		Type:           events.EventTypeSavedTrade,
		Payload:        payload,
	}
}

