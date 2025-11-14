package models

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// TradeValidationCheck represents validation results
type TradeValidationCheck struct {
	OK       bool
	Errors   []string
	Warnings []string
}

// ValidateTradeMath validates all math invariants in a trade
func ValidateTradeMath(trade Trade) TradeValidationCheck {
	errors := []string{}
	warnings := []string{}

	baseScale := assetScale(trade.Market.BaseAsset)
	quoteScale := assetScale(trade.Market.QuoteAsset)

	// 1) Per-fill math: price * quantity ≈ quoteAmount (within quantum)
	for _, fill := range trade.Fills {
		price, err := decimal.NewFromString(fill.Price)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Fill %s: invalid price format: %v", fill.TradeReference, err))
			continue
		}

		qty, err := decimal.NewFromString(fill.Quantity)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Fill %s: invalid quantity format: %v", fill.TradeReference, err))
			continue
		}

		got, err := decimal.NewFromString(fill.QuoteAmount)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Fill %s: invalid quoteAmount format: %v", fill.TradeReference, err))
			continue
		}

		expected := roundTo(price.Mul(qty), quoteScale)
		diff := expected.Sub(got).Abs()

		if diff.GreaterThan(quantum(quoteScale)) {
			errors = append(errors, fmt.Sprintf(
				"Fill %s: price*qty=%s vs quoteAmount=%s differs by %s > quantum(%d)",
				fill.TradeReference, expected.String(), got.String(), diff.String(), quoteScale))
		}
	}

	// 2) Executed sums
	sumQty := decimal.Zero
	sumQuote := decimal.Zero
	for _, fill := range trade.Fills {
		qty, err := decimal.NewFromString(fill.Quantity)
		if err != nil {
			continue // Already reported above
		}
		sumQty = sumQty.Add(qty)

		qa, err := decimal.NewFromString(fill.QuoteAmount)
		if err != nil {
			continue // Already reported above
		}
		sumQuote = sumQuote.Add(qa)
	}

	if trade.Executed.Quantity != nil && *trade.Executed.Quantity != "" {
		eq, err := decimal.NewFromString(*trade.Executed.Quantity)
		if err != nil {
			errors = append(errors, fmt.Sprintf("executed.quantity: invalid format: %v", err))
		} else if eq.Sub(sumQty).Abs().GreaterThan(quantum(baseScale)) {
			errors = append(errors, fmt.Sprintf(
				"executed.quantity %s != sum(fills.quantity) %s",
				eq.String(), sumQty.String()))
		}
	}

	if trade.Executed.QuoteAmount != nil && *trade.Executed.QuoteAmount != "" {
		eqa, err := decimal.NewFromString(*trade.Executed.QuoteAmount)
		if err != nil {
			errors = append(errors, fmt.Sprintf("executed.quoteAmount: invalid format: %v", err))
		} else if eqa.Sub(sumQuote).Abs().GreaterThan(quantum(quoteScale)) {
			errors = append(errors, fmt.Sprintf(
				"executed.quoteAmount %s != sum(fills.quoteAmount) %s",
				eqa.String(), sumQuote.String()))
		}
	}

	// 3) Average price
	if trade.Executed.AveragePrice != nil && *trade.Executed.AveragePrice != "" && !sumQty.IsZero() {
		computed := sumQuote.Div(sumQty)
		got, err := decimal.NewFromString(*trade.Executed.AveragePrice)
		if err != nil {
			errors = append(errors, fmt.Sprintf("executed.averagePrice: invalid format: %v", err))
		} else if !computed.Round(8).Equal(got.Round(8)) {
			warnings = append(warnings, fmt.Sprintf(
				"executed.averagePrice %s != computed %s (rounding differences are acceptable)",
				got.String(), computed.String()))
		}
	}

	// 4) Aggregated fees check
	if len(trade.Fees) > 0 {
		fromFills := sumFeesByAssetFromFills(trade)
		for _, fee := range trade.Fees {
			top, err := decimal.NewFromString(fee.Amount)
			if err != nil {
				errors = append(errors, fmt.Sprintf("aggregated fee %s: invalid amount format: %v", fee.Asset, err))
				continue
			}

			if v, ok := fromFills[fee.Asset]; ok {
				if !v.Equal(top) {
					warnings = append(warnings, fmt.Sprintf(
						"aggregated fee %s: top-level=%s vs sum(fills)=%s (ensure no double-counting)",
						fee.Asset, top.String(), v.String()))
				}
			}
		}
	}

	return TradeValidationCheck{
		OK:       len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}

// ExpectedLegAmounts calculates expected BASE and QUOTE payment amounts
func ExpectedLegAmounts(trade Trade) (base decimal.Decimal, quote decimal.Decimal) {
	sumQty := decimal.Zero
	sumQuote := decimal.Zero

	for _, fill := range trade.Fills {
		qty, err := decimal.NewFromString(fill.Quantity)
		if err != nil {
			continue
		}
		sumQty = sumQty.Add(qty)

		qa, err := decimal.NewFromString(fill.QuoteAmount)
		if err != nil {
			continue
		}
		sumQuote = sumQuote.Add(qa)
	}

	feeByAsset := sumFeesByAssetFromFills(trade)
	baseFee := feeByAsset[trade.Market.BaseAsset]
	quoteFee := feeByAsset[trade.Market.QuoteAsset]

	if trade.Side == TRADE_SIDE_BUY {
		return sumQty, sumQuote.Add(quoteFee)
	} else {
		return sumQty.Add(baseFee), sumQuote
	}
}

// ValidatePaymentsAgainstTrade validates that payments match trade expectations
func ValidatePaymentsAgainstTrade(trade Trade, basePayment Payment, quotePayment Payment) TradeValidationCheck {
	errors := []string{}
	warnings := []string{}

	baseAsset := trade.Market.BaseAsset
	quoteAsset := trade.Market.QuoteAsset

	if basePayment.Asset != baseAsset {
		errors = append(errors, fmt.Sprintf(
			"Base payment asset %s != %s", basePayment.Asset, baseAsset))
	}
	if quotePayment.Asset != quoteAsset {
		errors = append(errors, fmt.Sprintf(
			"Quote payment asset %s != %s", quotePayment.Asset, quoteAsset))
	}

	baseScale := assetScale(baseAsset)
	quoteScale := assetScale(quoteAsset)

	baseExp, quoteExp := ExpectedLegAmounts(trade)

	// Convert big.Int amounts to decimal for comparison
	baseGot := decimal.NewFromBigInt(basePayment.Amount, 0)
	quoteGot := decimal.NewFromBigInt(quotePayment.Amount, 0)

	// Check directions/types
	if trade.Side == TRADE_SIDE_BUY {
		if basePayment.Type != PAYMENT_TYPE_PAYIN {
			errors = append(errors, "BUY: base payment should be PAY-IN")
		}
		if quotePayment.Type != PAYMENT_TYPE_PAYOUT {
			errors = append(errors, "BUY: quote payment should be PAY-OUT")
		}
	} else {
		if basePayment.Type != PAYMENT_TYPE_PAYOUT {
			errors = append(errors, "SELL: base payment should be PAY-OUT")
		}
		if quotePayment.Type != PAYMENT_TYPE_PAYIN {
			errors = append(errors, "SELL: quote payment should be PAY-IN")
		}
	}

	// Check amounts within one quantum
	baseDiff := baseExp.Sub(baseGot).Abs()
	quoteDiff := quoteExp.Sub(quoteGot).Abs()

	if baseDiff.GreaterThan(quantum(baseScale)) {
		errors = append(errors, fmt.Sprintf(
			"Base payment amount %s != expected %s (diff %s)",
			baseGot.String(), baseExp.String(), baseDiff.String()))
	}

	if quoteDiff.GreaterThan(quantum(quoteScale)) {
		errors = append(errors, fmt.Sprintf(
			"Quote payment amount %s != expected %s (diff %s)",
			quoteGot.String(), quoteExp.String(), quoteDiff.String()))
	}

	return TradeValidationCheck{
		OK:       len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}

// Helper functions

// assetScale extracts the scale from an asset string (e.g., "USDC/6" -> 6)
func assetScale(asset string) int32 {
	parts := strings.Split(asset, "/")
	if len(parts) == 2 {
		scale, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0
		}
		return int32(scale)
	}
	return 0
}

// quantum returns the smallest unit for a given scale (10^-scale)
func quantum(scale int32) decimal.Decimal {
	ten := decimal.NewFromInt(10)
	return decimal.NewFromInt(1).Div(ten.Pow(decimal.NewFromInt(int64(scale))))
}

// roundTo rounds a decimal to the specified scale
func roundTo(d decimal.Decimal, scale int32) decimal.Decimal {
	return d.Round(scale)
}

// sumFeesByAssetFromFills sums all fees by asset from fills
func sumFeesByAssetFromFills(trade Trade) map[string]decimal.Decimal {
	out := map[string]decimal.Decimal{}
	for _, fill := range trade.Fills {
		for _, fee := range fill.Fees {
			d, err := decimal.NewFromString(fee.Amount)
			if err != nil {
				continue
			}

			if cur, ok := out[fee.Asset]; ok {
				out[fee.Asset] = cur.Add(d)
			} else {
				out[fee.Asset] = d
			}
		}
	}
	return out
}

// CreatePaymentsFromTrade creates two Payment objects from a Trade
// Returns (basePayment, quotePayment, error)
func CreatePaymentsFromTrade(trade Trade, portfolioAccountID AccountID) (Payment, Payment, error) {
	baseExp, quoteExp := ExpectedLegAmounts(trade)

	// Convert decimals to big.Int for Payment amounts
	baseAmount := new(big.Int)
	baseAmount.SetString(baseExp.String(), 10)

	quoteAmount := new(big.Int)
	quoteAmount.SetString(quoteExp.String(), 10)

	// Determine payment types based on trade side
	var baseType, quoteType PaymentType
	var baseSource, baseDest, quoteSource, quoteDest *AccountID

	if trade.Side == TRADE_SIDE_BUY {
		// BUY: receive base, spend quote
		baseType = PAYMENT_TYPE_PAYIN
		baseDest = &portfolioAccountID

		quoteType = PAYMENT_TYPE_PAYOUT
		quoteSource = &portfolioAccountID
	} else {
		// SELL: give base, receive quote
		baseType = PAYMENT_TYPE_PAYOUT
		baseSource = &portfolioAccountID

		quoteType = PAYMENT_TYPE_PAYIN
		quoteDest = &portfolioAccountID
	}

	// Create base payment
	basePaymentRef := fmt.Sprintf("trade:%s:BASE", trade.ID.String())
	basePayment := Payment{
		ID: PaymentID{
			PaymentReference: PaymentReference{
				Reference: basePaymentRef,
				Type:      baseType,
			},
			ConnectorID: trade.ConnectorID,
		},
		ConnectorID:          trade.ConnectorID,
		Reference:            basePaymentRef,
		CreatedAt:            trade.CreatedAt,
		Type:                 baseType,
		InitialAmount:        baseAmount,
		Amount:               baseAmount,
		Asset:                trade.Market.BaseAsset,
		Scheme:               PAYMENT_SCHEME_EXCHANGE,
		Status:               PAYMENT_STATUS_SUCCEEDED,
		SourceAccountID:      baseSource,
		DestinationAccountID: baseDest,
		Metadata: map[string]string{
			"tradeID": trade.ID.String(),
			"role":    "BASE",
			"market":  trade.Market.Symbol,
			"side":    trade.Side.String(),
		},
	}

	// Create quote payment
	quotePaymentRef := fmt.Sprintf("trade:%s:QUOTE", trade.ID.String())
	quotePayment := Payment{
		ID: PaymentID{
			PaymentReference: PaymentReference{
				Reference: quotePaymentRef,
				Type:      quoteType,
			},
			ConnectorID: trade.ConnectorID,
		},
		ConnectorID:          trade.ConnectorID,
		Reference:            quotePaymentRef,
		CreatedAt:            trade.CreatedAt,
		Type:                 quoteType,
		InitialAmount:        quoteAmount,
		Amount:               quoteAmount,
		Asset:                trade.Market.QuoteAsset,
		Scheme:               PAYMENT_SCHEME_EXCHANGE,
		Status:               PAYMENT_STATUS_SUCCEEDED,
		SourceAccountID:      quoteSource,
		DestinationAccountID: quoteDest,
		Metadata: map[string]string{
			"tradeID": trade.ID.String(),
			"role":    "QUOTE",
			"market":  trade.Market.Symbol,
			"side":    trade.Side.String(),
		},
	}

	// Create adjustments for each fill
	for _, fill := range trade.Fills {
		fillQty, _ := decimal.NewFromString(fill.Quantity)
		fillQuoteAmt, _ := decimal.NewFromString(fill.QuoteAmount)

		baseAdjAmount := new(big.Int)
		baseAdjAmount.SetString(fillQty.String(), 10)

		quoteAdjAmount := new(big.Int)
		quoteAdjAmount.SetString(fillQuoteAmt.String(), 10)

		baseAdjRef := fmt.Sprintf("fill:%s", fill.TradeReference)
		baseAdj := PaymentAdjustment{
			ID: PaymentAdjustmentID{
				PaymentID: basePayment.ID,
				Reference: baseAdjRef,
				CreatedAt: fill.Timestamp,
				Status:    PAYMENT_STATUS_SUCCEEDED,
			},
			Reference: baseAdjRef,
			CreatedAt: fill.Timestamp,
			Status:    PAYMENT_STATUS_SUCCEEDED,
			Amount:    baseAdjAmount,
			Asset:     &trade.Market.BaseAsset,
			Metadata:  map[string]string{},
			Raw:       fill.Raw,
		}

		quoteAdjRef := fmt.Sprintf("fill:%s:QUOTE", fill.TradeReference)
		quoteAdj := PaymentAdjustment{
			ID: PaymentAdjustmentID{
				PaymentID: quotePayment.ID,
				Reference: quoteAdjRef,
				CreatedAt: fill.Timestamp,
				Status:    PAYMENT_STATUS_SUCCEEDED,
			},
			Reference: quoteAdjRef,
			CreatedAt: fill.Timestamp,
			Status:    PAYMENT_STATUS_SUCCEEDED,
			Amount:    quoteAdjAmount,
			Asset:     &trade.Market.QuoteAsset,
			Metadata:  map[string]string{},
			Raw:       fill.Raw,
		}

		basePayment.Adjustments = append(basePayment.Adjustments, baseAdj)
		quotePayment.Adjustments = append(quotePayment.Adjustments, quoteAdj)
	}

	return basePayment, quotePayment, nil
}
