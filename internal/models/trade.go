package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// TradeMarket
type TradeMarket struct {
	Symbol     string `json:"symbol"`
	BaseAsset  string `json:"baseAsset"`
	QuoteAsset string `json:"quoteAsset"`
}

type TradeRequested struct {
	Quantity            *string `json:"quantity,omitempty"`
	LimitPrice          *string `json:"limitPrice,omitempty"`
	NotionalQuoteAmount *string `json:"notionalQuoteAmount,omitempty"`
	ClientOrderID       *string `json:"clientOrderID,omitempty"`
}

type TradeExecuted struct {
	Quantity     *string    `json:"quantity,omitempty"`
	QuoteAmount  *string    `json:"quoteAmount,omitempty"`
	AveragePrice *string    `json:"averagePrice,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

type TradeFee struct {
	Asset     string             `json:"asset"`
	Amount    string             `json:"amount"`
	Kind      *TradeFeeKind      `json:"kind,omitempty"`
	AppliedOn *TradeFeeAppliedOn `json:"appliedOn,omitempty"`
	Rate      *string            `json:"rate,omitempty"`
}

type TradeFill struct {
	TradeReference string           `json:"tradeReference"`
	Timestamp      time.Time        `json:"timestamp"`
	Price          string           `json:"price"`
	Quantity       string           `json:"quantity"`
	QuoteAmount    string           `json:"quoteAmount"`
	Liquidity      *TradeLiquidity  `json:"liquidity,omitempty"`
	Fees           []TradeFee       `json:"fees"`
	Raw            json.RawMessage  `json:"raw"`
}

// TradeLeg represents a payment leg linked to a trade
type TradeLeg struct {
	Role      TradeLegRole      `json:"role"`
	Direction TradeLegDirection `json:"direction"`
	Asset     string            `json:"asset"` // "CODE/scale" format
	NetAmount string            `json:"netAmount"`
	PaymentID *PaymentID        `json:"paymentID,omitempty"`
	Status    *PaymentStatus    `json:"status,omitempty"`
}

// Trade represents a trading transaction (exchange, stock, etc.)
type Trade struct {
	// Unique Trade ID generated from trade information
	ID TradeID `json:"id"`
	// Related Connector ID
	ConnectorID ConnectorID `json:"connectorID"`
	// PSP trade/transaction reference
	Reference string `json:"reference"`
	// Trade Creation date
	CreatedAt time.Time `json:"createdAt"`
	// Trade Update date
	UpdatedAt time.Time `json:"updatedAt"`

	// Optional portfolio/account this trade belongs to
	PortfolioAccountID *AccountID `json:"portfolioAccountID,omitempty"`

	// Instrument type: SPOT, FX, etc.
	InstrumentType TradeInstrumentType `json:"instrumentType"`
	// Execution model: ORDER_BOOK, RFQ, etc.
	ExecutionModel TradeExecutionModel `json:"executionModel"`

	// Market information
	Market TradeMarket `json:"market"`

	// Trade side: BUY or SELL
	Side TradeSide `json:"side"`
	// Order type: MARKET, LIMIT, etc.
	OrderType *TradeOrderType `json:"orderType,omitempty"`
	// Time in force: GTC, IOC, etc.
	TimeInForce *TradeTimeInForce `json:"timeInForce,omitempty"`
	// Trade status
	Status TradeStatus `json:"status"`

	// Requested parameters
	Requested TradeRequested `json:"requested"`
	// Executed results
	Executed TradeExecuted `json:"executed"`

	// Aggregated fees
	Fees []TradeFee `json:"fees"`
	// Individual fills
	Fills []TradeFill `json:"fills"`
	// Payment legs
	Legs []TradeLeg `json:"legs"`

	// Additional metadata
	Metadata map[string]string `json:"metadata"`
	// PSP response in raw
	Raw json.RawMessage `json:"raw"`
}

func (t Trade) MarshalJSON() ([]byte, error) {
	type Alias Trade

	return json.Marshal(&struct {
		ID          string `json:"id"`
		ConnectorID string `json:"connectorID"`
		Provider    string `json:"provider"`
		*Alias
	}{
		ID:          t.ID.String(),
		ConnectorID: t.ConnectorID.String(),
		Provider:    ToV3Provider(t.ConnectorID.Provider),
		Alias:       (*Alias)(&t),
	})
}

func (t *Trade) UnmarshalJSON(data []byte) error {
	type Alias Trade
	aux := &struct {
		ID          string `json:"id"`
		ConnectorID string `json:"connectorID"`
		Provider    string `json:"provider"` // Ignored, derived from ConnectorID
		*Alias
	}{
		Alias: (*Alias)(t),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	id, err := TradeIDFromString(aux.ID)
	if err != nil {
		return err
	}
	t.ID = id

	connectorID, err := ConnectorIDFromString(aux.ConnectorID)
	if err != nil {
		return err
	}
	t.ConnectorID = connectorID

	if t.PortfolioAccountID != nil {
		// Already unmarshaled correctly by Alias
	}

	return nil
}

func (t Trade) IdempotencyKey() string {
	return IdempotencyKey(t.ID)
}

// PSPTrade is the internal struct used by the plugins for ingestion
type PSPTrade struct {
	// PSP trade/transaction reference
	Reference string
	// Trade Creation date
	CreatedAt time.Time
	// Trade Update date (optional, defaults to CreatedAt if not set)
	UpdatedAt *time.Time

	// Optional portfolio/account reference
	PortfolioAccountReference *string

	// Instrument type: SPOT, FX, etc.
	InstrumentType TradeInstrumentType
	// Execution model: ORDER_BOOK, RFQ, etc.
	ExecutionModel TradeExecutionModel

	// Market information
	Market TradeMarket

	// Trade side: BUY or SELL
	Side TradeSide
	// Order type: MARKET, LIMIT, etc.
	OrderType *TradeOrderType
	// Time in force: GTC, IOC, etc.
	TimeInForce *TradeTimeInForce
	// Trade status
	Status TradeStatus

	// Requested parameters
	Requested TradeRequested
	// Executed results
	Executed TradeExecuted

	// Aggregated fees
	Fees []TradeFee
	// Individual fills
	Fills []TradeFill

	// Additional metadata
	Metadata map[string]string
	// PSP response in raw
	Raw json.RawMessage
}

func (p *PSPTrade) Validate() error {
	if p.Reference == "" {
		return fmt.Errorf("trade reference is required")
	}

	if p.CreatedAt.IsZero() {
		return fmt.Errorf("trade createdAt is required")
	}

	if p.Market.Symbol == "" {
		return fmt.Errorf("trade market symbol is required")
	}

	if p.Market.BaseAsset == "" {
		return fmt.Errorf("trade market baseAsset is required")
	}

	if p.Market.QuoteAsset == "" {
		return fmt.Errorf("trade market quoteAsset is required")
	}

	if len(p.Fills) == 0 {
		return fmt.Errorf("trade must have at least one fill")
	}

	return nil
}

func FromPSPTrade(from PSPTrade, connectorID ConnectorID) (Trade, error) {
	if err := from.Validate(); err != nil {
		return Trade{}, err
	}

	var portfolioAccountID *AccountID
	if from.PortfolioAccountReference != nil {
		id := AccountID{
			Reference:   *from.PortfolioAccountReference,
			ConnectorID: connectorID,
		}
		portfolioAccountID = &id
	}

	updatedAt := from.CreatedAt
	if from.UpdatedAt != nil {
		updatedAt = *from.UpdatedAt
	}

	return Trade{
		ID:                 TradeID_FromReference(from.Reference, connectorID),
		ConnectorID:        connectorID,
		Reference:          from.Reference,
		CreatedAt:          from.CreatedAt,
		UpdatedAt:          updatedAt,
		PortfolioAccountID: portfolioAccountID,
		InstrumentType:     from.InstrumentType,
		ExecutionModel:     from.ExecutionModel,
		Market:             from.Market,
		Side:               from.Side,
		OrderType:          from.OrderType,
		TimeInForce:        from.TimeInForce,
		Status:             from.Status,
		Requested:          from.Requested,
		Executed:           from.Executed,
		Fees:               from.Fees,
		Fills:              from.Fills,
		Metadata:           from.Metadata,
		Raw:                from.Raw,
	}, nil
}

func FromPSPTrades(from []PSPTrade, connectorID ConnectorID) ([]Trade, error) {
	trades := make([]Trade, 0, len(from))
	for _, t := range from {
		trade, err := FromPSPTrade(t, connectorID)
		if err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}
	return trades, nil
}
