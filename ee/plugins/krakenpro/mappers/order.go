package mappers

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

// PairResolution holds the (base, quote) split for a Kraken pair code.
// Looked up via the cached /0/public/AssetPairs map; falls back to
// matching on altname/wsname for codes the cache hasn't seen. BaseCode
// and QuoteCode are the raw Kraken codes (e.g. "XXBT", "ZUSD") — these
// are the per-variant spot account references; BaseSymbol/QuoteSymbol
// are their normalised tickers.
type PairResolution struct {
	Pair        string
	Wsname      string
	BaseSymbol  string
	QuoteSymbol string
	BaseCode    string
	QuoteCode   string
}

func pairResolution(code string, entry client.AssetPair) PairResolution {
	return PairResolution{
		Pair:        code,
		Wsname:      entry.Wsname,
		BaseSymbol:  NormalizeAsset(entry.Base),
		QuoteSymbol: NormalizeAsset(entry.Quote),
		BaseCode:    strings.ToUpper(strings.TrimSpace(entry.Base)),
		QuoteCode:   strings.ToUpper(strings.TrimSpace(entry.Quote)),
	}
}

// ResolvePair maps a Kraken pair code (e.g. "XXBTZUSD") to its
// (base, quote) tickers. `pairs` is the cached AssetPairs map keyed
// by the same code. Returns ok=false when neither the cache nor the
// wsname produces a usable split — the caller logs and skips.
func ResolvePair(pairs map[string]client.AssetPair, pair string) (PairResolution, bool) {
	if entry, ok := pairs[pair]; ok {
		return pairResolution(pair, entry), true
	}
	// Fallback: match any cached pair on its code, altname or wsname,
	// comparing slash-normalized so wsname forms like "XBT/USD" resolve.
	candidate := slashless(pair)
	for code, entry := range pairs {
		if slashless(code) == candidate ||
			slashless(entry.Altname) == candidate ||
			slashless(entry.Wsname) == candidate {
			return pairResolution(code, entry), true
		}
	}
	return PairResolution{}, false
}

// slashless upper-cases and removes "/" so pair codes, altnames and
// wsnames ("XBT/USD") compare on equal footing.
func slashless(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), "/", ""))
}

// OrderEntryWithID pairs an OrderEntry with its map-key id. The
// orchestrator gathers these from both OpenOrders and ClosedOrders
// before mapping; emission is one PSPOrder per id.
type OrderEntryWithID struct {
	OrderID string
	Order   client.OrderEntry
}

// maxPricePrecisionCap bounds the dynamic precision used for
// AverageFillPrice / LimitPrice to dodge float-noise digits Kraken's
// matching engine sometimes leaks (e.g. "27500.000000000001").
const maxPricePrecisionCap = 10

// orderAmounts bundles the *big.Int quantities parseOrderAmounts
// derives from one order row.
type orderAmounts struct {
	volOrdered, volExec   *big.Int
	cost, fee             *big.Int
	avgPrice              *big.Int
	limitPrice, stopPrice *big.Int
	pricePrecision        int
}

// OrderEntryToPSPOrder maps a single OpenOrders/ClosedOrders row to
// a PSPOrder. The order already carries cumulative state (vol_exec /
// cost / fee are running totals) — no aggregation across fills/pages,
// which keeps emissions faithful to the engine's adjustment dedup.
//
// Wallet resolution is best-effort: a leg whose asset has no current
// spot account (e.g. a historical order in a no-longer-held asset)
// gets a nil account reference rather than failing — the PSPOrder model
// permits nil source/destination refs (see MAPPINGS §8). The client-
// assigned cl_ord_id, when present, maps to ClientOrderID + metadata.
func OrderEntryToPSPOrder(
	currencies map[string]int,
	pairs map[string]client.AssetPair,
	row OrderEntryWithID,
) (*models.PSPOrder, error) {
	oe := row.Order

	pairRes, basePrec, quotePrec, err := resolveOrderPair(currencies, pairs, oe.Descr.Pair, row.OrderID)
	if err != nil {
		return nil, err
	}

	// BUY spends quote / receives base; SELL inverts. src = spent, dst = received.
	var direction models.OrderDirection
	srcSym, dstSym := pairRes.QuoteSymbol, pairRes.BaseSymbol
	srcCode, dstCode := pairRes.QuoteCode, pairRes.BaseCode
	switch strings.ToLower(oe.Descr.Type) {
	case "buy":
		direction = models.ORDER_DIRECTION_BUY
	case "sell":
		direction = models.ORDER_DIRECTION_SELL
		srcSym, dstSym = pairRes.BaseSymbol, pairRes.QuoteSymbol
		srcCode, dstCode = pairRes.BaseCode, pairRes.QuoteCode
	default:
		return nil, fmt.Errorf("unknown direction %q on order %s", oe.Descr.Type, row.OrderID)
	}

	amounts, err := parseOrderAmounts(oe, basePrec, quotePrec, row.OrderID)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(oe)
	if err != nil {
		return nil, fmt.Errorf("order %s marshal: %w", row.OrderID, err)
	}

	orderType, known := MapOrderType(oe.Descr.Ordertype)
	if !known {
		return nil, fmt.Errorf("unrecognized order type %q on order %s", oe.Descr.Ordertype, row.OrderID)
	}
	status, _ := MapOrderStatus(oe.Status, oe.Vol, oe.VolExec)
	quoteAsset := FormatAsset(currencies, pairRes.QuoteSymbol)
	priceAsset := fmt.Sprintf("%s/%d", pairRes.QuoteSymbol, amounts.pricePrecision)

	// CreatedAt is the order's open time and must stay stable across the
	// open->closed upsert: closed rows carry the same Reference, so deriving
	// it from closetm would mutate the creation timestamp on each refresh.
	// Close time is preserved in metadata instead.
	createdAt := FloatEpochToTime(oe.Opentm)

	metadata := OrderMetadata(oe.Descr.Pair, pairRes.Wsname, oe.Trades, oe.Descr.Ordertype, priceAsset)
	if oe.ClOrdID != "" {
		metadata[MetadataPrefix+"cl_ord_id"] = oe.ClOrdID
	}
	if oe.Closetm > 0 {
		metadata[MetadataPrefix+"close_time"] = FloatEpochToTime(oe.Closetm).Format(time.RFC3339)
	}

	return &models.PSPOrder{
		Reference:                   row.OrderID,
		ClientOrderID:               oe.ClOrdID,
		CreatedAt:                   createdAt,
		Direction:                   direction,
		Type:                        orderType,
		Status:                      status,
		SourceAsset:                 FormatAsset(currencies, srcSym),
		DestinationAsset:            FormatAsset(currencies, dstSym),
		BaseQuantityOrdered:         amounts.volOrdered,
		BaseQuantityFilled:          amounts.volExec,
		LimitPrice:                  amounts.limitPrice,
		StopPrice:                   amounts.stopPrice,
		QuoteAmount:                 amounts.cost,
		QuoteAsset:                  quoteAsset,
		AverageFillPrice:            amounts.avgPrice,
		Fee:                         amounts.fee,
		FeeAsset:                    &quoteAsset,
		PriceAsset:                  &priceAsset,
		TimeInForce:                 models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED,
		SourceAccountReference:      accountRef(srcCode),
		DestinationAccountReference: accountRef(dstCode),
		Metadata:                    metadata,
		Raw:                         raw,
	}, nil
}

// PairResolvable reports whether an order's pair resolves against the
// cache and both its assets have a known precision — i.e. the row can map
// without an error. Orchestrators use it to force one cache refresh before
// advancing past a pair listed after the last refresh.
func PairResolvable(currencies map[string]int, pairs map[string]client.AssetPair, descrPair string) bool {
	_, _, _, err := resolveOrderPair(currencies, pairs, descrPair, "")
	return err == nil
}

// resolveOrderPair resolves the (base, quote) symbols + their
// precisions from the cached AssetPairs map.
func resolveOrderPair(
	currencies map[string]int,
	pairs map[string]client.AssetPair,
	descrPair, orderID string,
) (PairResolution, int, int, error) {
	pairRes, ok := ResolvePair(pairs, descrPair)
	if !ok {
		return PairResolution{}, 0, 0, fmt.Errorf("unknown pair %q on order %s", descrPair, orderID)
	}
	basePrec, ok := currencies[pairRes.BaseSymbol]
	if !ok {
		return PairResolution{}, 0, 0, fmt.Errorf("unknown base asset %q on order %s", pairRes.BaseSymbol, orderID)
	}
	quotePrec, ok := currencies[pairRes.QuoteSymbol]
	if !ok {
		return PairResolution{}, 0, 0, fmt.Errorf("unknown quote asset %q on order %s", pairRes.QuoteSymbol, orderID)
	}
	return pairRes, basePrec, quotePrec, nil
}

// parseOrderAmounts converts the order's raw decimal-string fields
// into *big.Int minor-unit amounts at the right precisions. fee
// falls back to zero on parse failure to tolerate blank fields
// Kraken returns on canceled/expired rows.
func parseOrderAmounts(oe client.OrderEntry, basePrec, quotePrec int, orderID string) (orderAmounts, error) {
	a := orderAmounts{pricePrecision: dynamicOrderPricePrecision(oe.Descr.Price, oe.Descr.Price2, oe.Price)}

	var err error
	if a.volOrdered, err = ParseDecimalAmount(oe.Vol, basePrec); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s vol: %w", orderID, err)
	}
	if a.volExec, err = ParseDecimalAmount(orZero(oe.VolExec), basePrec); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s vol_exec: %w", orderID, err)
	}
	if a.cost, err = ParseDecimalAmount(orZero(oe.Cost), quotePrec); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s cost: %w", orderID, err)
	}
	// A blank fee (canceled/expired rows) is zero; a non-empty value that
	// won't parse is a real data error and must surface, not be hidden.
	if strings.TrimSpace(oe.Fee) == "" {
		a.fee = new(big.Int)
	} else if a.fee, err = ParseDecimalAmount(oe.Fee, quotePrec); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s fee: %w", orderID, err)
	}
	if a.avgPrice, err = parseOptionalPrice(oe.Price, a.pricePrecision); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s avg price: %w", orderID, err)
	}
	if a.limitPrice, err = parseOptionalPrice(oe.Descr.Price, a.pricePrecision); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s limit price: %w", orderID, err)
	}
	if a.stopPrice, err = parseOptionalPrice(oe.Descr.Price2, a.pricePrecision); err != nil {
		return orderAmounts{}, fmt.Errorf("order %s stop price: %w", orderID, err)
	}
	return a, nil
}

// MapOrderStatus mirrors coinbaseprime's mapCoinbaseStatus: Kraken's
// status enum is small (pending/open/closed/canceled/expired) and
// the FILLED/PARTIAL distinction is derived from the (vol_exec, vol)
// pair. Unknown values map to PENDING + known=false so the caller
// can log the gap.
func MapOrderStatus(status, vol, volExec string) (models.OrderStatus, bool) {
	filled := !IsZeroAmount(volExec)
	fullyFilled := filled && cmpDecimal(volExec, vol) >= 0
	partiallyFilled := filled && !fullyFilled

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return models.ORDER_STATUS_PENDING, true
	case "open":
		if partiallyFilled {
			return models.ORDER_STATUS_PARTIALLY_FILLED, true
		}
		return models.ORDER_STATUS_OPEN, true
	case "closed":
		switch {
		case fullyFilled:
			return models.ORDER_STATUS_FILLED, true
		case partiallyFilled:
			return models.ORDER_STATUS_PARTIALLY_FILLED, true
		default:
			// Closed without filling — treat as cancelled so the
			// state semantically reflects "no execution happened".
			return models.ORDER_STATUS_CANCELLED, true
		}
	case "canceled", "cancelled":
		if partiallyFilled {
			return models.ORDER_STATUS_PARTIALLY_FILLED, true
		}
		return models.ORDER_STATUS_CANCELLED, true
	case "expired":
		return models.ORDER_STATUS_EXPIRED, true
	default:
		return models.ORDER_STATUS_PENDING, false
	}
}

// cmpDecimal compares two decimal strings via big.Float to dodge the
// "vol=1.00000000 vs vol_exec=1" textual mismatch. Returns -1, 0, +1.
func cmpDecimal(a, b string) int {
	af, _, errA := big.ParseFloat(orZero(a), 10, 256, big.ToNearestEven)
	bf, _, errB := big.ParseFloat(orZero(b), 10, 256, big.ToNearestEven)
	if errA != nil || errB != nil {
		return 0
	}
	return af.Cmp(bf)
}

// dynamicOrderPricePrecision picks the largest fractional precision
// seen across the order's price fields, capped at maxPricePrecisionCap.
func dynamicOrderPricePrecision(prices ...string) int {
	max := 0
	for _, p := range prices {
		if i := strings.IndexByte(p, '.'); i >= 0 {
			d := len(p) - i - 1
			if d > max {
				max = d
			}
		}
	}
	if max > maxPricePrecisionCap {
		max = maxPricePrecisionCap
	}
	return max
}

func parseOptionalPrice(s string, precision int) (*big.Int, error) {
	if strings.TrimSpace(s) == "" || IsZeroAmount(s) {
		return nil, nil
	}
	return ParseDecimalAmount(s, precision)
}
