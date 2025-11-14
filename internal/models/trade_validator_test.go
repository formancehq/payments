package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTradeMath(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)

	result := models.ValidateTradeMath(trade)

	require.True(t, result.OK)
	require.Empty(t, result.Errors)
}

func TestValidateTradeMathFails(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	wrongQty := "2"
	trade.Executed.Quantity = &wrongQty

	result := models.ValidateTradeMath(trade)

	require.False(t, result.OK)
	require.NotEmpty(t, result.Errors)
}

func TestExpectedLegAmounts(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	base, quote := models.ExpectedLegAmounts(trade)

	assert.Equal(t, decimal.RequireFromString("1"), base)
	assert.Equal(t, decimal.RequireFromString("101.0"), quote)
}

func TestValidatePaymentsAgainstTrade(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	basePayment, quotePayment := newTestPayments(t, trade)

	result := models.ValidatePaymentsAgainstTrade(trade, basePayment, quotePayment)

	require.True(t, result.OK)
	require.Empty(t, result.Errors)
}

func TestCreatePaymentsFromTrade(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	accountID := *trade.PortfolioAccountID

	basePayment, quotePayment, err := models.CreatePaymentsFromTrade(trade, accountID)
	require.NoError(t, err)

	assert.Equal(t, models.PAYMENT_SCHEME_EXCHANGE, basePayment.Scheme)
	assert.Equal(t, models.PAYMENT_SCHEME_EXCHANGE, quotePayment.Scheme)
	assert.Len(t, basePayment.Adjustments, len(trade.Fills))
	assert.Len(t, quotePayment.Adjustments, len(trade.Fills))
}

func newTestTrade(side models.TradeSide) models.Trade {
	connectorID := models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "generic",
	}
	tradeID := models.TradeID_FromReference("trade-ref", connectorID)
	now := time.Now().UTC()
	quantity := "1"
	quoteAmount := "100.5"
	avgPrice := "100.5"

	return models.Trade{
		ID:          tradeID,
		ConnectorID: connectorID,
		Reference:   "trade-ref",
		CreatedAt:   now,
		UpdatedAt:   now,
		PortfolioAccountID: &models.AccountID{
			Reference:   "portfolio",
			ConnectorID: connectorID,
		},
		InstrumentType: models.TRADE_INSTRUMENT_TYPE_SPOT,
		ExecutionModel: models.TRADE_EXECUTION_MODEL_ORDER_BOOK,
		Market: models.TradeMarket{
			Symbol:     "BTC-USD",
			BaseAsset:  "BTC/8",
			QuoteAsset: "USD/2",
		},
		Side:   side,
		Status: models.TRADE_STATUS_FILLED,
		Requested: models.TradeRequested{
			Quantity: &quantity,
		},
		Executed: models.TradeExecuted{
			Quantity:     &quantity,
			QuoteAmount:  &quoteAmount,
			AveragePrice: &avgPrice,
			CompletedAt:  &now,
		},
		Fees: []models.TradeFee{
			{
				Asset:  "USD/2",
				Amount: "0.5",
				Kind:   pointerForTradeFeeKind(models.TRADE_FEE_KIND_TAKER),
				Rate:   pointerForString("0.0005"),
				AppliedOn: pointerForTradeFeeAppliedOn(
					models.TRADE_FEE_APPLIED_ON_QUOTE),
			},
		},
		Fills: []models.TradeFill{
			{
				TradeReference: "fill-1",
				Timestamp:      now,
				Price:          "100.5",
				Quantity:       "1",
				QuoteAmount:    "100.5",
				Fees: []models.TradeFee{
					{
						Asset:     "USD/2",
						Amount:    "0.5",
						Kind:      pointerForTradeFeeKind(models.TRADE_FEE_KIND_TAKER),
						AppliedOn: pointerForTradeFeeAppliedOn(models.TRADE_FEE_APPLIED_ON_QUOTE),
					},
				},
				Raw: json.RawMessage("{}"),
			},
		},
		Metadata: map[string]string{},
		Raw:      json.RawMessage("{}"),
	}
}

func newTestPayments(t *testing.T, trade models.Trade) (models.Payment, models.Payment) {
	t.Helper()

	accountID := *trade.PortfolioAccountID
	basePayment, quotePayment, err := models.CreatePaymentsFromTrade(trade, accountID)
	require.NoError(t, err)

	return basePayment, quotePayment
}

func pointerForTradeFeeKind(kind models.TradeFeeKind) *models.TradeFeeKind {
	return &kind
}

func pointerForTradeFeeAppliedOn(applied models.TradeFeeAppliedOn) *models.TradeFeeAppliedOn {
	return &applied
}

func pointerForString(val string) *string {
	return &val
}
