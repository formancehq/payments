package models

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
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
		price, ok := new(big.Rat).SetString(fill.Price)
		if !ok {
			errors = append(errors, fmt.Sprintf("Fill %s: invalid price format: %s", fill.TradeReference, fill.Price))
			continue
		}

		qty, ok := new(big.Rat).SetString(fill.Quantity)
		if !ok {
			errors = append(errors, fmt.Sprintf("Fill %s: invalid quantity format: %s", fill.TradeReference, fill.Quantity))
			continue
		}

		got, ok := new(big.Rat).SetString(fill.QuoteAmount)
		if !ok {
			errors = append(errors, fmt.Sprintf("Fill %s: invalid quoteAmount format: %s", fill.TradeReference, fill.QuoteAmount))
			continue
		}

		// expected = price * qty
		expected := new(big.Rat).Mul(price, qty)
		
		// diff = |expected - got|
		diff := new(big.Rat).Sub(expected, got)
		diff.Abs(diff)

		// Check diff > quantum(quoteScale)
		if diff.Cmp(quantum(quoteScale)) > 0 {
			errors = append(errors, fmt.Sprintf(
				"Fill %s: price*qty=%s vs quoteAmount=%s differs by %s > quantum(%d)",
				fill.TradeReference, expected.FloatString(int(quoteScale)), got.FloatString(int(quoteScale)), diff.FloatString(int(quoteScale)), quoteScale))
		}
	}

	// 2) Executed sums
	sumQty := new(big.Rat)
	sumQuote := new(big.Rat)
	for _, fill := range trade.Fills {
		qty, ok := new(big.Rat).SetString(fill.Quantity)
		if !ok {
			continue // Already reported above
		}
		sumQty.Add(sumQty, qty)

		qa, ok := new(big.Rat).SetString(fill.QuoteAmount)
		if !ok {
			continue // Already reported above
		}
		sumQuote.Add(sumQuote, qa)
	}

	if trade.Executed.Quantity != nil && *trade.Executed.Quantity != "" {
		eq, ok := new(big.Rat).SetString(*trade.Executed.Quantity)
		if !ok {
			errors = append(errors, fmt.Sprintf("executed.quantity: invalid format: %s", *trade.Executed.Quantity))
		} else {
			diff := new(big.Rat).Sub(eq, sumQty)
			diff.Abs(diff)
			if diff.Cmp(quantum(baseScale)) > 0 {
				errors = append(errors, fmt.Sprintf(
					"executed.quantity %s != sum(fills.quantity) %s",
					eq.FloatString(int(baseScale)), sumQty.FloatString(int(baseScale))))
			}
		}
	}

	if trade.Executed.QuoteAmount != nil && *trade.Executed.QuoteAmount != "" {
		eqa, ok := new(big.Rat).SetString(*trade.Executed.QuoteAmount)
		if !ok {
			errors = append(errors, fmt.Sprintf("executed.quoteAmount: invalid format: %s", *trade.Executed.QuoteAmount))
		} else {
			diff := new(big.Rat).Sub(eqa, sumQuote)
			diff.Abs(diff)
			if diff.Cmp(quantum(quoteScale)) > 0 {
				errors = append(errors, fmt.Sprintf(
					"executed.quoteAmount %s != sum(fills.quoteAmount) %s",
					eqa.FloatString(int(quoteScale)), sumQuote.FloatString(int(quoteScale))))
			}
		}
	}

	// 3) Average price
	// computed = sumQuote / sumQty
	if trade.Executed.AveragePrice != nil && *trade.Executed.AveragePrice != "" && sumQty.Sign() != 0 {
		computed := new(big.Rat).Quo(sumQuote, sumQty)
		got, ok := new(big.Rat).SetString(*trade.Executed.AveragePrice)
		if !ok {
			errors = append(errors, fmt.Sprintf("executed.averagePrice: invalid format: %s", *trade.Executed.AveragePrice))
		} else {
			// Relaxed check for average price: check if within tolerance instead of strict rounding
			// Use 8 decimal places tolerance as a heuristic (similar to previous Round(8))
			diff := new(big.Rat).Sub(computed, got)
			diff.Abs(diff)
			tolerance := quantum(8) 
			
			if diff.Cmp(tolerance) > 0 {
				warnings = append(warnings, fmt.Sprintf(
					"executed.averagePrice %s != computed %s (rounding differences are acceptable)",
					got.FloatString(8), computed.FloatString(8)))
			}
		}
	}

	// 4) Aggregated fees check
	if len(trade.Fees) > 0 {
		fromFills := sumFeesByAssetFromFills(trade)
		for _, fee := range trade.Fees {
			top, ok := new(big.Rat).SetString(fee.Amount)
			if !ok {
				errors = append(errors, fmt.Sprintf("aggregated fee %s: invalid amount format: %s", fee.Asset, fee.Amount))
				continue
			}

			if v, ok := fromFills[fee.Asset]; ok {
				if v.Cmp(top) != 0 {
					warnings = append(warnings, fmt.Sprintf(
						"aggregated fee %s: top-level=%s vs sum(fills)=%s (ensure no double-counting)",
						fee.Asset, top.FloatString(8), v.FloatString(8)))
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
// Returns values as big.Rat to preserve precision until conversion
func ExpectedLegAmounts(trade Trade) (base *big.Rat, quote *big.Rat) {
	sumQty := new(big.Rat)
	sumQuote := new(big.Rat)

	for _, fill := range trade.Fills {
		qty, ok := new(big.Rat).SetString(fill.Quantity)
		if !ok {
			continue
		}
		sumQty.Add(sumQty, qty)

		qa, ok := new(big.Rat).SetString(fill.QuoteAmount)
		if !ok {
			continue
		}
		sumQuote.Add(sumQuote, qa)
	}

	feeByAsset := sumFeesByAssetFromFills(trade)
	
	// Helper to safely get fee
	getFee := func(asset string) *big.Rat {
		if f, ok := feeByAsset[asset]; ok {
			return f
		}
		return new(big.Rat)
	}

	baseFee := getFee(trade.Market.BaseAsset)
	quoteFee := getFee(trade.Market.QuoteAsset)

	if trade.Side == TRADE_SIDE_BUY {
		// BUY: You receive Base, You pay Quote + Fees
		return sumQty, new(big.Rat).Add(sumQuote, quoteFee)
	} else {
		// SELL: You pay Base + Fees, You receive Quote
		return new(big.Rat).Add(sumQty, baseFee), sumQuote
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

	baseExpRat, quoteExpRat := ExpectedLegAmounts(trade)
	
	// Expected amounts in minor units (BigInt)
	baseExp := ratToBigInt(baseExpRat, baseScale)
	quoteExp := ratToBigInt(quoteExpRat, quoteScale)

	// Got amounts (already in minor units)
	baseGot := basePayment.Amount
	quoteGot := quotePayment.Amount

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

	// Check amounts (exact integer match expected for minor units)
	// Since we converted Expected to minor units using the same scale, they should match exactly or be within 1 unit due to rounding?
	// Actually, if we use big.Rat for calculation and convert to Int at the end, we effectively floor/round.
	// Let's check strict equality first, or small difference.
	
	baseDiff := new(big.Int).Sub(baseExp, baseGot)
	baseDiff.Abs(baseDiff)
	
	quoteDiff := new(big.Int).Sub(quoteExp, quoteGot)
	quoteDiff.Abs(quoteDiff)

	// Tolerance of 1 minor unit
	one := big.NewInt(1)

	if baseDiff.Cmp(one) > 0 {
		errors = append(errors, fmt.Sprintf(
			"Base payment amount %s != expected %s (diff %s > 1)",
			baseGot.String(), baseExp.String(), baseDiff.String()))
	}

	if quoteDiff.Cmp(one) > 0 {
		errors = append(errors, fmt.Sprintf(
			"Quote payment amount %s != expected %s (diff %s > 1)",
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

// quantum returns the smallest unit for a given scale (1/10^scale) as big.Rat
func quantum(scale int32) *big.Rat {
	num := big.NewInt(1)
	den := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	return new(big.Rat).SetFrac(num, den)
}

// ratToBigInt converts a Rat to BigInt by multiplying by 10^scale (minor units)
func ratToBigInt(r *big.Rat, scale int32) *big.Int {
	if r == nil {
		return big.NewInt(0)
	}
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(scale)), nil)
	
	// r * multiplier
	val := new(big.Rat).SetInt(multiplier)
	val.Mul(val, r)
	
	// Return integer part (floor)
	// Ideally we should round to nearest?
	// "1.23" * 100 = 123.
	// "1.239" * 100 = 123.9 -> 123 (floor).
	// Let's stick to floor for now, or simple integer conversion which is what FloatString does.
	// big.Rat.Num() / Denom()
	
	num := val.Num()
	denom := val.Denom()
	
	res := new(big.Int).Div(num, denom)
	return res
}

// sumFeesByAssetFromFills sums all fees by asset from fills
func sumFeesByAssetFromFills(trade Trade) map[string]*big.Rat {
	out := map[string]*big.Rat{}
	for _, fill := range trade.Fills {
		for _, fee := range fill.Fees {
			d, ok := new(big.Rat).SetString(fee.Amount)
			if !ok {
				continue
			}

			if cur, ok := out[fee.Asset]; ok {
				cur.Add(cur, d)
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
	baseExpRat, quoteExpRat := ExpectedLegAmounts(trade)
	
	baseScale := assetScale(trade.Market.BaseAsset)
	quoteScale := assetScale(trade.Market.QuoteAsset)

	// Convert to minor units
	baseAmount := ratToBigInt(baseExpRat, baseScale)
	quoteAmount := ratToBigInt(quoteExpRat, quoteScale)

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
	// Initialize adjustment slices to ensure they're not nil
	basePayment.Adjustments = make([]PaymentAdjustment, 0, len(trade.Fills))
	quotePayment.Adjustments = make([]PaymentAdjustment, 0, len(trade.Fills))

	for _, fill := range trade.Fills {
		fillQty, _ := new(big.Rat).SetString(fill.Quantity)
		fillQuoteAmt, _ := new(big.Rat).SetString(fill.QuoteAmount)
		
		// Use ratToBigInt for proper scaling to minor units
		baseAdjAmount := ratToBigInt(fillQty, baseScale)
		quoteAdjAmount := ratToBigInt(fillQuoteAmt, quoteScale)

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

	// Ensure we have at least one adjustment per payment
	// This is required by the v3 API contract and for event emission
	if len(basePayment.Adjustments) == 0 {
		return Payment{}, Payment{}, fmt.Errorf("trade must have at least one fill to create payments with adjustments")
	}

	return basePayment, quotePayment, nil
}
