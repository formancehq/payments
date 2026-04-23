package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

// resolveWallets builds a fresh symbol→walletID map from the engine's
// AccountLookup each time FetchNextOrders runs. The lookup reads from the
// persisted accounts table scoped to the connector, so the map is always
// up-to-date and consistent across pods — unlike the previous in-process
// cache which only reflected pages processed on the current pod.
//
// Only TRADING wallets are indexed: Coinbase Prime paginates TRADING /
// VAULT / ONCHAIN / QC / WALLET_TYPE_OTHER separately (see GET /wallets
// `type` query param — https://docs.cdp.coinbase.com/prime/reference/primerestapi_getportfoliowallets),
// so a single symbol (e.g. "USD") can back multiple accounts with the same
// DefaultAsset. Orders debit/credit only the TRADING wallet (see the
// `source_type` / `destination_type` fields on create-order —
// https://docs.cdp.coinbase.com/prime/reference/primerestapi_createorder),
// so non-TRADING rows must be excluded — otherwise a VAULT or ONCHAIN row
// could stomp the TRADING row in the map and resolution would return the
// wrong account.
//
// Returns a hard error if the engine never injected a lookup (configuration
// bug) or if the DB read fails.
func (p *Plugin) resolveWallets(ctx context.Context) (map[string]string, error) {
	if p.accountLookup == nil {
		return nil, fmt.Errorf("account lookup not wired: engine misconfiguration")
	}

	accounts, err := p.accountLookup.ListAccountsByConnector(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing accounts: %w", err)
	}

	wallets := make(map[string]string, len(accounts))
	for _, a := range accounts {
		if a.DefaultAsset == nil {
			continue
		}
		if a.Metadata["wallet_type"] != walletTypeTrading {
			continue
		}
		// DefaultAsset is in the form "SYMBOL/precision" (e.g. "BTC/8").
		symbol, _, ok := strings.Cut(*a.DefaultAsset, "/")
		if !ok {
			continue
		}
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			continue
		}
		wallets[symbol] = a.Reference
	}
	return wallets, nil
}

func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	var oldState incrementalState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	wallets, err := p.resolveWallets(ctx)
	if err != nil {
		return models.FetchNextOrdersResponse{}, err
	}

	ordersResp, err := p.client.ListOrders(ctx, oldState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to list orders: %w", err)
	}

	pspOrders := make([]models.PSPOrder, 0, len(ordersResp.Orders))
	for _, order := range ordersResp.Orders {
		pspOrder, err := p.clientOrderToPSPOrder(ctx, order, wallets)
		if err != nil {
			// Fail the batch when any order in the page references an
			// unresolved wallet (or anything else goes wrong). This is the
			// safe default: the cursor is not advanced, the page is
			// retried, and no order is dropped. This allows Temporal
			// retries, e.g. once a missing account is fetched during
			// fetchNextAccounts.
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to convert order %s: %w", order.ID, err)
		}
		pspOrders = append(pspOrders, pspOrder)
	}

	newState := incrementalState{Cursor: advanceCursor(oldState.Cursor, ordersResp.Pagination.NextCursor)}

	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	return models.FetchNextOrdersResponse{
		Orders:   pspOrders,
		NewState: stateBytes,
		HasMore:  ordersResp.Pagination.HasNext,
	}, nil
}

func (p *Plugin) clientOrderToPSPOrder(ctx context.Context, order client.Order, wallets map[string]string) (models.PSPOrder, error) {
	raw, err := json.Marshal(order)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to marshal order raw: %w", err)
	}

	parts := strings.Split(order.ProductID, "-")
	if len(parts) != 2 {
		return models.PSPOrder{}, fmt.Errorf("invalid product ID: %s", order.ProductID)
	}
	baseSymbol, quoteSymbol := parts[0], parts[1]

	baseAsset, _, baseOk, err := p.resolveAssetAndPrecision(ctx, baseSymbol)
	if err != nil {
		return models.PSPOrder{}, err
	}
	if !baseOk {
		return models.PSPOrder{}, fmt.Errorf("unsupported base asset: %s", baseSymbol)
	}
	quoteAsset, _, quoteOk, err := p.resolveAssetAndPrecision(ctx, quoteSymbol)
	if err != nil {
		return models.PSPOrder{}, err
	}
	if !quoteOk {
		return models.PSPOrder{}, fmt.Errorf("unsupported quote asset: %s", quoteSymbol)
	}

	// Pull a single snapshot of currencies to thread through parseOrderQuantity /
	// parseFeeFields. getAssets is cheap on the fresh-cache fast path (the two
	// resolveAssetAndPrecision calls above just refreshed the TTL).
	currencies, _, err := p.getAssets(ctx)
	if err != nil {
		return models.PSPOrder{}, err
	}

	quoteWallet := wallets[strings.ToUpper(quoteSymbol)]
	baseWallet := wallets[strings.ToUpper(baseSymbol)]
	if quoteWallet == "" {
		return models.PSPOrder{}, fmt.Errorf("unresolved wallet for quote symbol %q on order %s (will retry after next accounts cycle)", quoteSymbol, order.ID)
	}
	if baseWallet == "" {
		return models.PSPOrder{}, fmt.Errorf("unresolved wallet for base symbol %q on order %s (will retry after next accounts cycle)", baseSymbol, order.ID)
	}

	// BUY BTC-USD: source=USD (spend), target=BTC (receive)
	// SELL BTC-USD: source=BTC (spend), target=USD (receive)
	direction := models.ORDER_DIRECTION_BUY
	sourceAsset, destinationAsset := quoteAsset, baseAsset
	if strings.ToUpper(order.Side) == "SELL" {
		direction = models.ORDER_DIRECTION_SELL
		sourceAsset, destinationAsset = baseAsset, quoteAsset
	}

	orderType, knownType := mapCoinbaseOrderType(order.Type)
	if !knownType {
		p.logger.Infof("unknown coinbase order type %q for order %s, defaulting to MARKET", order.Type, order.ID)
	}
	status, knownStatus := mapCoinbaseStatus(order.Status, order.BaseQuantity, order.FilledQuantity)
	if !knownStatus {
		p.logger.Infof("unknown coinbase order status %q for order %s, defaulting to PENDING", order.Status, order.ID)
	}
	timeInForce, _ := models.TimeInForceFromString(strings.ToUpper(order.TimeInForce))
	if timeInForce == models.TIME_IN_FORCE_UNKNOWN {
		timeInForce = models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	}

	baseQuantityOrdered, err := parseOrderQuantity(currencies, order.BaseQuantity, baseSymbol)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse base quantity: %w", err)
	}

	baseQuantityFilled, err := parseOrderQuantity(currencies, order.FilledQuantity, baseSymbol)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse filled quantity: %w", err)
	}

	// Quote amount (filled_value) at quote precision — the exact USD amount
	quoteAmount, err := parseOrderQuantity(currencies, order.FilledValue, quoteSymbol)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse filled value: %w", err)
	}

	// Fee at quote precision
	fee, feeAsset, err := parseFeeFields(currencies, order, quoteSymbol, quoteAsset)
	if err != nil {
		return models.PSPOrder{}, err
	}

	// Use dynamic precision: max decimal places across all price fields for this
	// order, so all prices are at the same scale and directly comparable.
	pricePrecision := maxPricePrecision(order.LimitPrice, order.StopPrice, order.AverageFilledPrice)

	limitPrice, err := p.parseOptionalPrice(order.LimitPrice, pricePrecision)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse limit price: %w", err)
	}

	stopPrice, err := p.parseOptionalPrice(order.StopPrice, pricePrecision)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse stop price: %w", err)
	}

	avgFillPrice, err := p.parseOptionalPrice(order.AverageFilledPrice, pricePrecision)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse average filled price: %w", err)
	}

	createdAt, err := time.Parse(time.RFC3339, order.CreatedAt)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse order createdAt %q: %w", order.CreatedAt, err)
	}

	var expiresAt *time.Time
	if order.ExpiryTime != "" {
		t, err := time.Parse(time.RFC3339, order.ExpiryTime)
		if err != nil {
			return models.PSPOrder{}, fmt.Errorf("failed to parse order expiryTime %q: %w", order.ExpiryTime, err)
		}
		expiresAt = &t
	}

	metadata := p.buildOrderMetadata(order, baseSymbol, quoteSymbol, pricePrecision, wallets)
	priceAsset := fmt.Sprintf("%s/%d", quoteSymbol, pricePrecision)

	// Resolve account references from wallet map. Both wallet IDs were
	// validated non-empty at the top of this function, so they are guaranteed
	// to resolve here.
	// BUY: source = quote wallet (USD out), destination = base wallet (crypto in)
	// SELL: source = base wallet (crypto out), destination = quote wallet (USD in)
	var sourceAccountRef, destAccountRef *string
	if direction == models.ORDER_DIRECTION_BUY {
		sourceAccountRef = &quoteWallet
		destAccountRef = &baseWallet
	} else {
		sourceAccountRef = &baseWallet
		destAccountRef = &quoteWallet
	}

	return models.PSPOrder{
		Reference:                   order.ID,
		ClientOrderID:               order.ClientOrderID,
		CreatedAt:                   createdAt,
		Direction:                   direction,
		Type:                        orderType,
		SourceAsset:                 sourceAsset,
		DestinationAsset:            destinationAsset,
		BaseQuantityOrdered:         baseQuantityOrdered,
		BaseQuantityFilled:          baseQuantityFilled,
		LimitPrice:                  limitPrice,
		StopPrice:                   stopPrice,
		QuoteAmount:                 quoteAmount,
		QuoteAsset:                  quoteAsset,
		AverageFillPrice:            avgFillPrice,
		Fee:                         fee,
		FeeAsset:                    feeAsset,
		PriceAsset:                  &priceAsset,
		SourceAccountReference:      sourceAccountRef,
		DestinationAccountReference: destAccountRef,
		Status:                      status,
		TimeInForce:                 timeInForce,
		ExpiresAt:                   expiresAt,
		Metadata:                    metadata,
		Raw:                         raw,
	}, nil
}

func mapCoinbaseOrderType(t string) (models.OrderType, bool) {
	switch strings.ToUpper(t) {
	case "MARKET":
		return models.ORDER_TYPE_MARKET, true
	case "LIMIT":
		return models.ORDER_TYPE_LIMIT, true
	case "STOP_LIMIT":
		return models.ORDER_TYPE_STOP_LIMIT, true
	case "TWAP":
		return models.ORDER_TYPE_TWAP, true
	case "VWAP":
		return models.ORDER_TYPE_VWAP, true
	case "PEG":
		return models.ORDER_TYPE_PEG, true
	case "BLOCK":
		return models.ORDER_TYPE_BLOCK, true
	case "RFQ":
		return models.ORDER_TYPE_RFQ, true
	default:
		return models.ORDER_TYPE_MARKET, false
	}
}

func parseFeeFields(currencies map[string]int, order client.Order, quoteSymbol, quoteAsset string) (*big.Int, *string, error) {
	if order.Commission == "" {
		return nil, nil, nil
	}
	fee, err := parseOrderQuantity(currencies, order.Commission, quoteSymbol)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse commission: %w", err)
	}
	return fee, &quoteAsset, nil
}

// parseOptionalPrice parses price fields that may contain float artifacts from
// the exchange. Uses parseDecimalString (truncating) instead of go-libs (strict)
// because Coinbase returns prices like "1825.6099999998417653".
func (p *Plugin) parseOptionalPrice(value string, precision int) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	return parseDecimalString(value, precision)
}

// parseDecimalString converts a decimal string to *big.Int using pure string
// manipulation -- no float64 anywhere -- truncating excess decimals to handle
// float artifacts from exchange APIs.
func parseDecimalString(value string, precision int) (*big.Int, error) {
	if value == "" {
		return nil, fmt.Errorf("empty decimal string")
	}

	// Strip leading/trailing whitespace
	value = strings.TrimSpace(value)

	parts := strings.SplitN(value, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	if len(fracPart) > precision {
		fracPart = fracPart[:precision]
	} else {
		fracPart += strings.Repeat("0", precision-len(fracPart))
	}

	result, ok := new(big.Int).SetString(intPart+fracPart, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal: %s", value)
	}
	return result, nil
}

// decimalPlaces returns the number of digits after the decimal point in a string.
func decimalPlaces(s string) int {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return len(s) - i - 1
	}
	return 0
}

// maxPricePrecision returns the max decimal places across non-empty values,
// capped at 10. Coinbase returns prices computed with float64, which can
// produce strings like "1825.6099999998417653" -- the trailing digits past
// ~10 decimals are float noise, not real precision.
const maxPricePrecisionCap = 10

func maxPricePrecision(values ...string) int {
	m := 0
	for _, v := range values {
		if d := decimalPlaces(v); d > m {
			m = d
		}
	}
	if m > maxPricePrecisionCap {
		m = maxPricePrecisionCap
	}
	return m
}

// buildOrderMetadata captures valuable Coinbase Prime fields that don't map to
// structured Order fields, preserving data that's useful for reconciliation and debugging.
func (p *Plugin) buildOrderMetadata(order client.Order, baseSymbol, quoteSymbol string, pricePrecision int, wallets map[string]string) map[string]string {
	m := make(map[string]string)

	set := func(k, v string) {
		if v != "" {
			m[MetadataPrefix+k] = v
		}
	}

	set("product_id", order.ProductID)
	set("portfolio_id", order.PortfolioID)
	set("client_order_id", order.ClientOrderID)
	set("quote_value", order.QuoteValue)
	set("filled_value", order.FilledValue)
	set("order_total", order.OrderTotal)
	set("exchange_fee", order.ExchangeFee)
	set("net_average_filled_price", order.NetAverageFilledPrice)
	set("historical_pov", order.HistoricalPov)
	set("quote_currency", quoteSymbol)
	m[MetadataPrefix+"price_asset"] = fmt.Sprintf("%s/%d", quoteSymbol, pricePrecision)

	set("base_wallet_id", wallets[strings.ToUpper(baseSymbol)])
	set("quote_wallet_id", wallets[strings.ToUpper(quoteSymbol)])

	if order.CommissionDetail != nil {
		set("commission_total", order.CommissionDetail.TotalCommission)
		set("commission_client", order.CommissionDetail.ClientCommission)
		set("commission_venue", order.CommissionDetail.VenueCommission)
		set("commission_ces", order.CommissionDetail.CesCommission)
		set("commission_financing", order.CommissionDetail.FinancingCommission)
		set("commission_regulatory", order.CommissionDetail.RegulatoryCommission)
		set("commission_clearing", order.CommissionDetail.ClearingCommission)
	}

	if order.PostOnly {
		m[MetadataPrefix+"post_only"] = "true"
	}

	return m
}

func mapCoinbaseStatus(cbStatus, baseQuantity, filledQuantity string) (models.OrderStatus, bool) {
	switch strings.ToUpper(cbStatus) {
	case "PENDING":
		return models.ORDER_STATUS_PENDING, true
	case "OPEN":
		if filledQuantity != "" && filledQuantity != "0" {
			filled, ok1 := new(big.Float).SetString(filledQuantity)
			base, ok2 := new(big.Float).SetString(baseQuantity)
			if ok1 && ok2 && filled.Sign() > 0 && filled.Cmp(base) < 0 {
				return models.ORDER_STATUS_PARTIALLY_FILLED, true
			}
		}
		return models.ORDER_STATUS_OPEN, true
	case "FILLED":
		return models.ORDER_STATUS_FILLED, true
	case "CANCELLED":
		return models.ORDER_STATUS_CANCELLED, true
	case "EXPIRED":
		return models.ORDER_STATUS_EXPIRED, true
	case "FAILED":
		return models.ORDER_STATUS_FAILED, true
	default:
		return models.ORDER_STATUS_PENDING, false
	}
}

// parseOrderQuantity converts a string quantity to a big.Int using the
// precision lookup from the provided currencies snapshot. Unknown assets
// fall back to a default precision of 8 (matching the pre-refactor
// behavior). Callers are responsible for sourcing currencies via
// p.getAssets(ctx) — keeping the parameter explicit avoids the implicit
// "must-be-fresh" contract the previous p.currencies read had.
func parseOrderQuantity(currencies map[string]int, quantityStr string, asset string) (*big.Int, error) {
	if quantityStr == "" {
		return big.NewInt(0), nil
	}
	return currency.GetAmountWithPrecisionFromString(quantityStr, precisionFor(currencies, asset))
}

func precisionFor(currencies map[string]int, asset string) int {
	if currencies != nil {
		if precision, ok := currencies[strings.ToUpper(asset)]; ok {
			return precision
		}
	}
	// Default precision for unknown assets
	return 8
}
