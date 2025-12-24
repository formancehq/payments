package models_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTradeMath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*models.Trade)
		wantErr bool
	}{
		{
			name:    "valid trade",
			mutate:  func(t *models.Trade) {},
			wantErr: false,
		},
		{
			name: "invalid quantity",
			mutate: func(t *models.Trade) {
				q := "2"
				t.Executed.Quantity = &q
			},
			wantErr: true,
		},
		{
			name: "invalid quote amount",
			mutate: func(t *models.Trade) {
				qa := "200.5"
				t.Executed.QuoteAmount = &qa
			},
			wantErr: true,
		},
		{
			name: "invalid fill price math",
			mutate: func(t *models.Trade) {
				// 100.5 * 1 = 100.5, but we say quote amount is 200
				t.Fills[0].QuoteAmount = "200"
			},
			wantErr: true,
		},
		{
			name: "invalid fee summation",
			mutate: func(t *models.Trade) {
				// Fee on fill is 0.5, top level fee claims 1.0
				t.Fees[0].Amount = "1.0"
			},
			wantErr: false, // This is a warning, not an error
		},
		{
			name: "average price mismatch",
			mutate: func(t *models.Trade) {
				// 100.5 / 1 = 100.5, but we say avg price is 200
				p := "200"
				t.Executed.AveragePrice = &p
			},
			wantErr: false, // This is a warning
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			trade := newTestTrade(models.TRADE_SIDE_BUY)
			tt.mutate(&trade)

			result := models.ValidateTradeMath(trade)
			if tt.wantErr {
				require.False(t, result.OK, "expected validation failure but got OK")
				require.NotEmpty(t, result.Errors, "expected validation errors")
			} else {
				require.True(t, result.OK, fmt.Sprintf("expected validation success but got errors: %v", result.Errors))
			}
		})
	}
}

func TestValidateTradeMathRounding(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	
	// Case: 1.000001 * 100 = 100.0001
	// Quote asset is USD/2 (2 decimals)
	// 100.0001 should round to 100.00
	
	trade.Fills[0].Price = "100"
	trade.Fills[0].Quantity = "1.000001"
	trade.Fills[0].QuoteAmount = "100.00" // Exact rounded value

	// Update executed values to match sum of fills
	q := "1.000001"
	qa := "100.00"
	trade.Executed.Quantity = &q
	trade.Executed.QuoteAmount = &qa

	result := models.ValidateTradeMath(trade)
	require.True(t, result.OK, "rounding should be handled correctly")
	require.Empty(t, result.Errors)
}

func TestExpectedLegAmounts(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	base, quote := models.ExpectedLegAmounts(trade)

	// Expected: Base 1, Quote 100.5 + 0.5 = 101.0
	
	baseExpected, _ := new(big.Rat).SetString("1")
	quoteExpected, _ := new(big.Rat).SetString("101.0")

	assert.Equal(t, 0, base.Cmp(baseExpected))
	assert.Equal(t, 0, quote.Cmp(quoteExpected))
}

func TestCreatePaymentsFromTradePrecision(t *testing.T) {
	t.Parallel()

	trade := newTestTrade(models.TRADE_SIDE_BUY)
	// Set up a high precision scenario
	// Base: BTC/8 -> 1.12345678
	// Quote: USD/2 -> price 50000.00
	
	trade.Market.BaseAsset = "BTC/8"
	trade.Market.QuoteAsset = "USD/2"
	
	qty := "1.12345678"
	price := "50000"
	// 1.12345678 * 50000 = 56172.839
	quoteAmt := "56172.839" 
	
	trade.Fills[0].Quantity = qty
	trade.Fills[0].Price = price
	trade.Fills[0].QuoteAmount = quoteAmt
	
	// Update fees to be consistent (0 fee for simplicity here)
	trade.Fills[0].Fees = nil
	trade.Fees = nil
	
	accountID := *trade.PortfolioAccountID
	basePayment, quotePayment, err := models.CreatePaymentsFromTrade(trade, accountID)
	require.NoError(t, err)
	
	// Verify Base Amount (BTC/8)
	// 1.12345678 * 10^8 = 112345678
	expectedBase := big.NewInt(112345678)
	assert.Equal(t, expectedBase, basePayment.Amount, "Base amount should preserve 8 decimals")
	
	// Verify Quote Amount (USD/2)
	// 56172.839 * 10^2 = 5617283.9 -> 5617283 (floor/truncate)
	expectedQuote := big.NewInt(5617283)
	assert.Equal(t, expectedQuote, quotePayment.Amount, "Quote amount should be converted to minor units correctly")
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
