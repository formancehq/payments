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

// TrackedOrderInput is the slim first-sight capture from open_orders/.
// Only the original limit Price needs persisting; market / type /
// subtype / datetime / amount_remaining come back live on each
// order_status/ poll. Empty Price (a MARKET order that filled in
// one cycle) → PSPOrder.LimitPrice = nil.
type TrackedOrderInput struct {
	Price       string
	FirstSeenAt time.Time
}

type OrderMapInput struct {
	Status           client.OrderStatus
	Tracked          TrackedOrderInput
	RetentionExpired bool
}

// OrderStatusToPSPOrder maps a Bitstamp order_status response to a
// PSPOrder. Self-trade fills are deduplicated by tid before
// aggregation. See MAPPINGS §4.4.
func OrderStatusToPSPOrder(currencies map[string]int, in OrderMapInput) (*models.PSPOrder, error) {
	pair := strings.ToLower(strings.TrimSpace(in.Status.Market))
	if pair == "" {
		return nil, fmt.Errorf("order %s: missing market on order_status response", in.Status.ID)
	}
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

	direction := OrderTypeStringToDirection(in.Status.Type)
	if direction == models.ORDER_DIRECTION_UNKNOWN {
		return nil, fmt.Errorf("order %s: unknown direction %q on order_status response", in.Status.ID, in.Status.Type)
	}

	orderType := OrderSubtypeToType(in.Status.Subtype)
	tif := OrderSubtypeToTIF(in.Status.Subtype)

	baseFilled, quoteFilled, totalFee, fillCount, err := aggregateFills(in.Status.Transactions, base, quote, basePrec, quotePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s fills: %w", in.Status.ID, err)
	}

	baseQuantityOrdered, err := computeBaseQuantityOrdered(baseFilled, in.Status.AmountRemaining, basePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s base quantity ordered: %w", in.Status.ID, err)
	}

	// LimitPrice is only relevant on resting / triggered LIMIT
	// orders; MARKET / INSTANT orders have no limit. Emit nil when
	// no first-sight value was captured (MARKET that filled within
	// one cycle).
	var limitPrice *big.Int
	if in.Tracked.Price != "" {
		parsed, perr := ParseAmount(in.Tracked.Price, quotePrec)
		if perr != nil {
			return nil, fmt.Errorf("order %s limit price: %w", in.Status.ID, perr)
		}
		limitPrice = parsed
	}

	createdAt := orderCreatedAt(in.Status.Datetime, in.Tracked.FirstSeenAt)
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
		CreatedAt:                   createdAt,
		Direction:                   direction,
		Type:                        orderType,
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
		TimeInForce:                 tif,
		SourceAccountReference:      strPtr(src),
		DestinationAccountReference: strPtr(dst),
		Metadata:                    orderMetadataFor(in, pair),
		Raw:                         raw,
	}, nil
}

// orderMetadataFor preserves order_subtype + order_status_datetime
// so downstream consumers can disambiguate MARKET vs INSTANT (both
// map to ORDER_TYPE_MARKET) and audit the wire timestamp.
func orderMetadataFor(in OrderMapInput, pair string) map[string]string {
	m := OrderMetadata(pair, in.Status.ClientOrderID, in.RetentionExpired)
	m[MetadataKeyOrderType] = strings.ToLower(in.Status.Subtype)
	if in.Status.Subtype != "" {
		m[MetadataKeyOrderSubtype] = in.Status.Subtype
	}
	if in.Status.Datetime != "" {
		m[MetadataKeyOrderDatetime] = in.Status.Datetime
	}
	return m
}

// orderCreatedAt prefers the wire datetime; falls back to FirstSeenAt
// when the wire value is missing or unparseable so the order stays
// emittable rather than failing the cycle.
func orderCreatedAt(wireDatetime string, firstSeenAt time.Time) time.Time {
	if wireDatetime == "" {
		return firstSeenAt
	}
	t, err := ParseBitstampTime(wireDatetime)
	if err != nil {
		return firstSeenAt
	}
	return t
}

// computeBaseQuantityOrdered = filled + amount_remaining when both
// are known. Returns nil (not zero) when amount_remaining is absent
// so PSPOrder.BaseQuantityOrdered surfaces unknown honestly.
func computeBaseQuantityOrdered(baseFilled *big.Int, amountRemaining string, basePrec int) (*big.Int, error) {
	if amountRemaining == "" {
		return nil, nil
	}
	remaining, err := ParseAmount(amountRemaining, basePrec)
	if err != nil {
		return nil, fmt.Errorf("amount_remaining %q: %w", amountRemaining, err)
	}
	return new(big.Int).Add(baseFilled, remaining), nil
}

// splitCurrencyPair parses Bitstamp's "<base><quote>" (lowercase) or
// "<base>/<quote>" pair notation into its (base, quote) tickers. The
// /all/ snapshot uses the concatenated form (e.g. "btcusd"); the
// order_status response uses the slash form ("BTC/USD"). Both are
// accepted.
//
// splitCurrencyPair parses "BTC/USD" (slash form) or "btcusd"
// (concat form). 3-letter base ticker + 3-or-4-letter quote.
func splitCurrencyPair(pair string) (base, quote string, err error) {
	if pair == "" {
		return "", "", fmt.Errorf("empty currency pair")
	}
	if idx := strings.IndexByte(pair, '/'); idx > 0 {
		return strings.ToUpper(pair[:idx]), strings.ToUpper(pair[idx+1:]), nil
	}
	switch len(pair) {
	case 6, 7, 8:
		return strings.ToUpper(pair[:3]), strings.ToUpper(pair[3:]), nil
	default:
		return "", "", fmt.Errorf("cannot split currency pair %q (length %d)", pair, len(pair))
	}
}

// accountReferencesForDirection: BUY → (quote, base), SELL → (base, quote).
func accountReferencesForDirection(d models.OrderDirection, base, quote string) (string, string) {
	if d == models.ORDER_DIRECTION_SELL {
		return base, quote
	}
	return quote, base
}

// aggregateFills sums per-fill base/quote/fee. Self-trade rows
// (duplicate tid) are deduplicated before summing.
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

// averageFillPrice = QuoteAmount × 10^basePrec / BaseQuantityFilled
// at quote precision. Returns zero (not nil) on no fills so callers
// can always dereference; "zero" means "not yet filled".
func averageFillPrice(quote, base *big.Int, basePrec, _ int) *big.Int {
	if base.Sign() == 0 {
		return new(big.Int)
	}
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
