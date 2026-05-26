package mappers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// SplitCurrencyPair parses Bitstamp's "<base><quote>" (lowercase) or
// "<base>/<quote>" pair notation into its (base, quote) tickers. The
// /all/ snapshot uses the concatenated form (e.g. "btcusd"); the
// order_status response uses the slash form ("BTC/USD"). Both are
// accepted.
//
// SplitCurrencyPair parses "BTC/USD" (slash form) or "btcusd"
// (concat form). 3-letter base ticker + 3-or-4-letter quote.
func SplitCurrencyPair(pair string) (base, quote string, err error) {
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

// AccountOrderDataEventToPSPOrder maps one account_order_data event to a
// PSPOrder. The market parameter is the lowercase pair symbol (e.g. "btcusd")
// and must match the value used to query the endpoint.
func AccountOrderDataEventToPSPOrder(currencies map[string]int, market string, event client.AccountOrderDataEvent) (*models.PSPOrder, error) {
	pair := strings.ToLower(strings.TrimSpace(market))
	base, quote, err := SplitCurrencyPair(pair)
	if err != nil {
		return nil, fmt.Errorf("order %s: %w", event.Data.IDStr, err)
	}
	basePrec, err := PrecisionFor(currencies, base)
	if err != nil {
		return nil, fmt.Errorf("order %s base: %w", event.Data.IDStr, err)
	}
	quotePrec, err := PrecisionFor(currencies, quote)
	if err != nil {
		return nil, fmt.Errorf("order %s quote: %w", event.Data.IDStr, err)
	}

	direction := OrderTypeIntToDirection(event.Data.OrderType)
	if direction == models.ORDER_DIRECTION_UNKNOWN {
		return nil, fmt.Errorf("order %s: unknown direction %d", event.Data.IDStr, event.Data.OrderType)
	}

	orderType := OrderSubtypeToType(event.Data.OrderSubtype)
	tif := OrderSubtypeToTIF(event.Data.OrderSubtype)

	baseQuantityOrdered, err := ParseDecimalAmount(event.Data.AmountAtCreate, basePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s amount_at_create: %w", event.Data.IDStr, err)
	}

	baseQuantityFilled, err := ParseDecimalAmount(event.Data.AmountTraded, basePrec)
	if err != nil {
		return nil, fmt.Errorf("order %s amount_traded: %w", event.Data.IDStr, err)
	}

	var limitPrice *big.Int
	if !IsZeroAmount(event.Data.PriceStr) {
		limitPrice, err = parsePriceToMinorUnits(event.Data.PriceStr, quotePrec)
		if err != nil {
			return nil, fmt.Errorf("order %s price: %w", event.Data.IDStr, err)
		}
	}

	quoteAmount := approxQuoteAmountFromPrice(baseQuantityFilled, limitPrice, basePrec)
	status := AccountOrderEventToStatus(event.Event, event.Data.AmountStr, event.Data.AmountTraded)
	createdAt := parseUnixMicrosecondsStr(event.Data.Microtimestamp)

	src, dst := accountReferencesForDirection(direction, base, quote)
	quoteAsset := FormatAsset(currencies, quote)

	raw, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("order %s marshal raw: %w", event.Data.IDStr, err)
	}

	return &models.PSPOrder{
		Reference:                   event.Data.IDStr,
		CreatedAt:                   createdAt,
		Direction:                   direction,
		Type:                        orderType,
		Status:                      status,
		SourceAsset:                 FormatAsset(currencies, src),
		DestinationAsset:            FormatAsset(currencies, dst),
		BaseQuantityOrdered:         baseQuantityOrdered,
		BaseQuantityFilled:          baseQuantityFilled,
		LimitPrice:                  limitPrice,
		QuoteAmount:                 quoteAmount,
		QuoteAsset:                  quoteAsset,
		PriceAsset:                  &quoteAsset,
		TimeInForce:                 tif,
		SourceAccountReference:      pointer.For(src),
		DestinationAccountReference: pointer.For(dst),
		Metadata:                    accountOrderEventMetadata(event, market, pair),
		Raw:                         raw,
	}, nil
}

// parsePriceToMinorUnits converts a price string (which may be in scientific
// notation like "7.74E+4") to integer minor units at the given precision.
// big.Float is used so that scientific notation and arbitrary decimal
// representations are all handled uniformly.
func parsePriceToMinorUnits(priceStr string, precision int) (*big.Int, error) {
	f, ok := new(big.Float).SetPrec(256).SetString(priceStr)
	if !ok {
		return nil, fmt.Errorf("invalid price %q", priceStr)
	}
	// Text('f', precision) always produces exactly `precision` decimal places,
	// which ParseDecimalAmount can handle directly.
	dec := f.Text('f', precision)
	return ParseDecimalAmount(dec, precision)
}

// approxQuoteAmountFromPrice estimates the quote amount as baseFilled × price.
// Both inputs are already in minor units:
//
//	quoteMinor = baseFilled × priceMinor / 10^basePrec
func approxQuoteAmountFromPrice(baseFilled, priceMinor *big.Int, basePrec int) *big.Int {
	if baseFilled == nil || baseFilled.Sign() == 0 || priceMinor == nil || priceMinor.Sign() == 0 {
		return new(big.Int)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(basePrec)), nil)
	return new(big.Int).Quo(new(big.Int).Mul(baseFilled, priceMinor), scale)
}

// parseUnixMicrosecondsStr parses a Unix-microseconds string into a UTC time.
// Returns zero time on any error.
func parseUnixMicrosecondsStr(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	us, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(us/1_000_000, (us%1_000_000)*1_000).UTC()
}

func accountOrderEventMetadata(
	event client.AccountOrderDataEvent,
	marketType string,
	pair string,
) map[string]string {
	m := map[string]string{
		MetadataKeyMarketType:   marketType,
		MetadataKeyCurrencyPair: pair,
	}
	setIfNonEmpty(m, MetadataKeyOrderEventType, event.Event)
	setIfNonEmpty(m, MetadataKeyOrderEventID, event.EventID)
	setIfNonEmpty(m, MetadataKeyOrderDatetimeSecs, event.Data.Datetime)
	setIfNonEmpty(m, MetadataKeyOrderDatetimeMicros, event.Data.Microtimestamp)
	return m
}
