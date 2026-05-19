package mappers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// TrackedOrderInput captures the original order parameters obtained
// from open_orders/ at first sight. order_status/ does NOT return
// price/amount/type/currency_pair — they MUST be supplied by the
// caller (sourced from ordersState.TrackedOrders[id]).
//
// Defined here so the state struct in the parent package can build a
// value of this type without re-declaring the schema; the mapper
// remains the source of truth for what the orchestrator must persist.
type TrackedOrderInput struct {
	Price        string
	Amount       string
	CurrencyPair string
	Type         int // 0 = BUY, 1 = SELL
	FirstSeenAt  time.Time
}

// OrderMapInput bundles every piece of state required to translate a
// Bitstamp order_status response into a PSPOrder.
//
//   - Status — the order_status payload (id + status + fills).
//   - Tracked — the first-sight capture from open_orders/.
//   - RetentionExpired — set true on the forced-final emit triggered
//     by the 25-day TrackedOrders eviction policy; surfaces as the
//     com.bitstamp.spec/retention_expired metadata flag.
type OrderMapInput struct {
	Status           client.OrderStatus
	Tracked          TrackedOrderInput
	RetentionExpired bool
}

// OrderStatusToPSPOrder maps a Bitstamp order_status response (plus
// the first-sight TrackedOrderInput) to a PSPOrder.
//
// Field derivations follow MAPPINGS.md §3.4.3. Self-trade fills are
// deduplicated by tid before aggregation so paired buy/sell legs on
// the same parent order are not double-counted.
func OrderStatusToPSPOrder(currencies map[string]int, in OrderMapInput) (*models.PSPOrder, error) {
	pair := strings.ToLower(strings.TrimSpace(in.Tracked.CurrencyPair))
	base, quote, err := splitCurrencyPair(pair)
	if err != nil {
		return nil, fmt.Errorf("order %s: %w", in.Status.ID, err)
	}
	basePrec, err := PrecisionFor(currencies, base)
	if err != nil {
		return nil, fmt.Errorf("order %s base: %w", in.Status.ID, err)
	}
	quotePrec, err := PrecisionFor(currencies, quote)
	if err != nil {
		return nil, fmt.Errorf("order %s quote: %w", in.Status.ID, err)
	}

	direction := OrderTypeIntToDirection(in.Tracked.Type)
	if direction == models.ORDER_DIRECTION_UNKNOWN {
		return nil, fmt.Errorf("order %s: unknown direction %d", in.Status.ID, in.Tracked.Type)
	}

	baseQuantityOrdered, err := ParseAmount(in.Tracked.Amount, basePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s base quantity: %w", in.Status.ID, err)
	}

	limitPrice, err := ParseAmount(in.Tracked.Price, quotePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s limit price: %w", in.Status.ID, err)
	}

	baseFilled, quoteFilled, totalFee, fillCount, err := aggregateFills(in.Status.Transactions, base, quote, basePrec, quotePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s fills: %w", in.Status.ID, err)
	}

	avgFillPrice := averageFillPrice(quoteFilled, baseFilled, basePrec, quotePrec)
	status := OrderStatusToPSPStatus(in.Status.Status, fillCount)

	src, dst := accountReferencesForDirection(direction, base, quote)
	quoteAsset := FormatAsset(currencies, quote)

	raw, err := json.Marshal(in.Status)
	if err != nil {
		return nil, fmt.Errorf("order %s marshal raw: %w", in.Status.ID, err)
	}

	feeAsset := quoteAsset
	return &models.PSPOrder{
		Reference:                   in.Status.ID.String(),
		ClientOrderID:               in.Status.ClientOrderID,
		CreatedAt:                   in.Tracked.FirstSeenAt,
		Direction:                   direction,
		// Bitstamp's open_orders/ snapshot only returns resting limit
		// orders — market orders are matched on the same call that
		// places them and never appear here.
		Type:                        models.ORDER_TYPE_LIMIT,
		Status:                      status,
		SourceAsset:                 FormatAsset(currencies, src),
		DestinationAsset:            FormatAsset(currencies, dst),
		BaseQuantityOrdered:         baseQuantityOrdered,
		BaseQuantityFilled:          baseFilled,
		LimitPrice:                  limitPrice,
		QuoteAmount:                 quoteFilled,
		QuoteAsset:                  quoteAsset,
		Fee:                         totalFee,
		FeeAsset:                    &feeAsset,
		AverageFillPrice:            avgFillPrice,
		PriceAsset:                  &quoteAsset,
		// Bitstamp limit orders have no explicit TIF; the platform
		// behaviour is good-till-cancelled.
		TimeInForce:                 models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		SourceAccountReference:      strPtr(src),
		DestinationAccountReference: strPtr(dst),
		Metadata:                    OrderMetadata(pair, in.Status.ClientOrderID, in.RetentionExpired),
		Raw:                         raw,
	}, nil
}

// splitCurrencyPair parses Bitstamp's "<base><quote>" (lowercase) or
// "<base>/<quote>" pair notation into its (base, quote) tickers. The
// /all/ snapshot uses the concatenated form (e.g. "btcusd"); we
// accept both for forward-compatibility.
//
// Bitstamp documented pair codes are 3-letter base ticker + 3 or 4
// letter quote ticker. We attempt the 3+3 split first and fall back
// to 3+4 (e.g. "btcusdc") so USDC pairs work without per-pair
// hard-coding.
func splitCurrencyPair(pair string) (base, quote string, err error) {
	if pair == "" {
		return "", "", fmt.Errorf("empty currency pair")
	}
	if idx := strings.IndexByte(pair, '/'); idx > 0 {
		return strings.ToUpper(pair[:idx]), strings.ToUpper(pair[idx+1:]), nil
	}
	switch {
	case len(pair) == 6:
		return strings.ToUpper(pair[:3]), strings.ToUpper(pair[3:]), nil
	case len(pair) == 7:
		return strings.ToUpper(pair[:3]), strings.ToUpper(pair[3:]), nil
	case len(pair) == 8:
		// e.g. btcusdc, ethusdc, xrpusdc — 3+4
		return strings.ToUpper(pair[:3]), strings.ToUpper(pair[3:]), nil
	default:
		return "", "", fmt.Errorf("cannot split currency pair %q (length %d)", pair, len(pair))
	}
}

// accountReferencesForDirection returns the (source, destination)
// account tickers per the convention in MAPPINGS.md §3.4:
//   - BUY:  source = quote, destination = base
//   - SELL: source = base,  destination = quote
func accountReferencesForDirection(d models.OrderDirection, base, quote string) (string, string) {
	if d == models.ORDER_DIRECTION_SELL {
		return base, quote
	}
	return quote, base
}

// aggregateFills sums the per-fill base / quote amounts and the fee
// from an order_status.transactions[] list. Self-trade rows (rows
// with duplicate tid) are deduplicated before summing.
func aggregateFills(fills []client.OrderTransaction, base, quote string, basePrec, quotePrec int) (baseFilled, quoteFilled, totalFee *big.Int, fillCount int, err error) {
	baseFilled = new(big.Int)
	quoteFilled = new(big.Int)
	totalFee = new(big.Int)

	seen := make(map[int64]struct{}, len(fills))
	for _, f := range fills {
		if f.TID > 0 {
			if _, dup := seen[f.TID]; dup {
				continue
			}
			seen[f.TID] = struct{}{}
		}
		fillCount++

		if baseAmt, ok := f.CurrencyAmounts[strings.ToLower(base)]; ok && !IsZeroAmount(baseAmt) {
			amt, perr := ParseAmount(AbsAmount(baseAmt), basePrec)
			if perr != nil {
				return nil, nil, nil, 0, fmt.Errorf("fill %d base %s: %w", f.TID, base, perr)
			}
			baseFilled.Add(baseFilled, amt)
		}
		if quoteAmt, ok := f.CurrencyAmounts[strings.ToLower(quote)]; ok && !IsZeroAmount(quoteAmt) {
			amt, perr := ParseAmount(AbsAmount(quoteAmt), quotePrec)
			if perr != nil {
				return nil, nil, nil, 0, fmt.Errorf("fill %d quote %s: %w", f.TID, quote, perr)
			}
			quoteFilled.Add(quoteFilled, amt)
		}
		if !IsZeroAmount(f.Fee) {
			fee, perr := ParseAmount(AbsAmount(f.Fee), quotePrec)
			if perr != nil {
				return nil, nil, nil, 0, fmt.Errorf("fill %d fee: %w", f.TID, perr)
			}
			totalFee.Add(totalFee, fee)
		}
	}
	return baseFilled, quoteFilled, totalFee, fillCount, nil
}

// averageFillPrice computes QuoteAmount * 10^pricePrecision /
// BaseQuantityFilled at the quote precision. Returns a zero big.Int
// (not nil) when no fills are present so the PSPOrder field is always
// safe to dereference; downstream the AverageFillPrice analytic field
// is treated as "zero == not-yet-filled".
func averageFillPrice(quote, base *big.Int, basePrec, quotePrec int) *big.Int {
	if base.Sign() == 0 {
		return new(big.Int)
	}
	// Scale numerator: quote (already at quotePrec) × 10^basePrec
	// so the division yields a value at quotePrec (i.e. denominated
	// in quote currency minor units per 1 base unit).
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(basePrec)), nil)
	num := new(big.Int).Mul(quote, scale)
	return new(big.Int).Quo(num, base)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
