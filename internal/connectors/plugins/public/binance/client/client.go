package client

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	binance "github.com/binance/binance-connector-go"
)

const (
	defaultTimeout = 30 * time.Second
	baseURL        = "https://api.binance.com"
	testnetBaseURL = "https://testnet.binance.vision"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	// Account operations
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)

	// Order operations
	GetOpenOrders(ctx context.Context, symbol string) ([]Order, error)
	GetAllOrders(ctx context.Context, params ListOrdersParams) ([]Order, error)
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, symbol string, orderID int64) (*CancelOrderResponse, error)

	// Exchange info
	GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error)

	// Market data operations (public, no auth required)
	GetOrderBook(ctx context.Context, symbol string, limit int) (*OrderBookResponse, error)
	GetTicker24hr(ctx context.Context, symbol string) (*TickerResponse, error)
	GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]KlineEntry, error)
}

type client struct {
	sdkClient *binance.Client
}

func New(
	connectorName string,
	apiKey string,
	secretKey string,
	testNet bool,
) Client {
	endpoint := baseURL
	if testNet {
		endpoint = testnetBaseURL
	}

	sdkClient := binance.NewClient(apiKey, secretKey, endpoint)

	return &client{
		sdkClient: sdkClient,
	}
}

func (c *client) GetAccountInfo(ctx context.Context) (*AccountInfo, error) {
	resp, err := c.sdkClient.NewGetAccountService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	balances := make([]Balance, len(resp.Balances))
	for i, b := range resp.Balances {
		balances[i] = Balance{
			Asset:  b.Asset,
			Free:   b.Free,
			Locked: b.Locked,
		}
	}

	return &AccountInfo{
		MakerCommission:  int(resp.MakerCommission),
		TakerCommission:  int(resp.TakerCommission),
		BuyerCommission:  int(resp.BuyerCommission),
		SellerCommission: int(resp.SellerCommission),
		CanTrade:         resp.CanTrade,
		CanWithdraw:      resp.CanWithdraw,
		CanDeposit:       resp.CanDeposit,
		UpdateTime:       int64(resp.UpdateTime),
		AccountType:      resp.AccountType,
		Balances:         balances,
	}, nil
}

func (c *client) GetOpenOrders(ctx context.Context, symbol string) ([]Order, error) {
	svc := c.sdkClient.NewGetOpenOrdersService()
	if symbol != "" {
		svc.Symbol(symbol)
	}

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	orders := make([]Order, len(resp))
	for i, o := range resp {
		orders[i] = Order{
			Symbol:             o.Symbol,
			OrderID:            o.OrderId,
			OrderListID:        o.OrderListId,
			ClientOrderID:      o.ClientOrderId,
			Price:              o.Price,
			OrigQty:            o.OrigQty,
			ExecutedQty:        o.ExecutedQty,
			CumulativeQuoteQty: o.CummulativeQuoteQty,
			Status:             o.Status,
			TimeInForce:        o.TimeInForce,
			Type:               o.Type,
			Side:               o.Side,
			StopPrice:          o.StopPrice,
			IcebergQty:         o.IcebergQty,
			Time:               int64(o.Time),
			UpdateTime:         int64(o.UpdateTime),
			IsWorking:          o.IsWorking,
			OrigQuoteOrderQty:  o.OrigQuoteOrderQty,
		}
	}

	return orders, nil
}

func (c *client) GetAllOrders(ctx context.Context, params ListOrdersParams) ([]Order, error) {
	svc := c.sdkClient.NewGetAllOrdersService().Symbol(params.Symbol)

	if params.OrderID > 0 {
		svc.OrderId(params.OrderID)
	}
	if !params.StartTime.IsZero() {
		svc.StartTime(uint64(params.StartTime.UnixMilli()))
	}
	if !params.EndTime.IsZero() {
		svc.EndTime(uint64(params.EndTime.UnixMilli()))
	}
	if params.Limit > 0 {
		svc.Limit(params.Limit)
	}

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders: %w", err)
	}

	orders := make([]Order, len(resp))
	for i, o := range resp {
		orders[i] = Order{
			Symbol:             o.Symbol,
			OrderID:            o.OrderId,
			OrderListID:        o.OrderListId,
			ClientOrderID:      o.ClientOrderId,
			Price:              o.Price,
			OrigQty:            o.OrigQty,
			ExecutedQty:        o.ExecutedQty,
			CumulativeQuoteQty: o.CummulativeQuoteQty,
			Status:             o.Status,
			TimeInForce:        o.TimeInForce,
			Type:               o.Type,
			Side:               o.Side,
			StopPrice:          o.StopPrice,
			IcebergQty:         o.IcebergQty,
			Time:               int64(o.Time),
			UpdateTime:         int64(o.UpdateTime),
			IsWorking:          o.IsWorking,
			OrigQuoteOrderQty:  o.OrigQuoteOrderQty,
		}
	}

	return orders, nil
}

func (c *client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	svc := c.sdkClient.NewCreateOrderService().
		Symbol(req.Symbol).
		Side(req.Side).
		Type(req.Type)

	if req.TimeInForce != "" {
		svc.TimeInForce(req.TimeInForce)
	}
	if req.Quantity != "" {
		qty, err := strconv.ParseFloat(req.Quantity, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity: %w", err)
		}
		svc.Quantity(qty)
	}
	if req.QuoteOrderQty != "" {
		qty, err := strconv.ParseFloat(req.QuoteOrderQty, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid quote order qty: %w", err)
		}
		svc.QuoteOrderQty(qty)
	}
	if req.Price != "" {
		price, err := strconv.ParseFloat(req.Price, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
		svc.Price(price)
	}
	if req.NewClientOrderID != "" {
		svc.NewClientOrderId(req.NewClientOrderID)
	}
	if req.StopPrice != "" {
		stopPrice, err := strconv.ParseFloat(req.StopPrice, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid stop price: %w", err)
		}
		svc.StopPrice(stopPrice)
	}
	if req.IcebergQty != "" {
		icebergQty, err := strconv.ParseFloat(req.IcebergQty, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid iceberg qty: %w", err)
		}
		svc.IcebergQuantity(icebergQty)
	}

	// Request FULL response type to get all details
	svc.NewOrderRespType("FULL")

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// The SDK returns interface{} because response type varies
	// We requested FULL, so we can type assert to CreateOrderResponseFULL
	fullResp, ok := resp.(*binance.CreateOrderResponseFULL)
	if !ok {
		// Try RESULT type as fallback
		resultResp, ok := resp.(*binance.CreateOrderResponseRESULT)
		if ok {
			return &CreateOrderResponse{
				Symbol:             resultResp.Symbol,
				OrderID:            resultResp.OrderId,
				OrderListID:        resultResp.OrderListId,
				ClientOrderID:      resultResp.ClientOrderId,
				TransactTime:       int64(resultResp.TransactTime),
				Price:              resultResp.Price,
				OrigQty:            resultResp.OrigQty,
				ExecutedQty:        resultResp.ExecutedQty,
				CumulativeQuoteQty: resultResp.CummulativeQuoteQty,
				Status:             resultResp.Status,
				TimeInForce:        resultResp.TimeInForce,
				Type:               resultResp.Type,
				Side:               resultResp.Side,
			}, nil
		}

		// Try ACK type as fallback
		ackResp, ok := resp.(*binance.CreateOrderResponseACK)
		if ok {
			return &CreateOrderResponse{
				Symbol:        ackResp.Symbol,
				OrderID:       ackResp.OrderId,
				OrderListID:   ackResp.OrderListId,
				ClientOrderID: ackResp.ClientOrderId,
				TransactTime:  int64(ackResp.TransactTime),
			}, nil
		}

		return nil, fmt.Errorf("unexpected response type from create order")
	}

	return &CreateOrderResponse{
		Symbol:             fullResp.Symbol,
		OrderID:            fullResp.OrderId,
		OrderListID:        fullResp.OrderListId,
		ClientOrderID:      fullResp.ClientOrderId,
		TransactTime:       int64(fullResp.TransactTime),
		Price:              fullResp.Price,
		OrigQty:            fullResp.OrigQty,
		ExecutedQty:        fullResp.ExecutedQty,
		CumulativeQuoteQty: fullResp.CummulativeQuoteQty,
		Status:             fullResp.Status,
		TimeInForce:        fullResp.TimeInForce,
		Type:               fullResp.Type,
		Side:               fullResp.Side,
	}, nil
}

func (c *client) CancelOrder(ctx context.Context, symbol string, orderID int64) (*CancelOrderResponse, error) {
	resp, err := c.sdkClient.NewCancelOrderService().
		Symbol(symbol).
		OrderId(orderID).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &CancelOrderResponse{
		Symbol:             resp.Symbol,
		OrigClientOrderID:  resp.OrigClientOrderId,
		OrderID:            resp.OrderId,
		OrderListID:        resp.OrderListId,
		ClientOrderID:      resp.ClientOrderId,
		Price:              resp.Price,
		OrigQty:            resp.OrigQty,
		ExecutedQty:        resp.ExecutedQty,
		CumulativeQuoteQty: resp.CummulativeQuoteQty,
		Status:             resp.Status,
		TimeInForce:        resp.TimeInForce,
		Type:               resp.Type,
		Side:               resp.Side,
	}, nil
}

func (c *client) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	resp, err := c.sdkClient.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}

	rateLimits := make([]RateLimit, len(resp.RateLimits))
	for i, r := range resp.RateLimits {
		rateLimits[i] = RateLimit{
			RateLimitType: r.RateLimitType,
			Interval:      r.Interval,
			IntervalNum:   1, // SDK doesn't provide this, default to 1
			Limit:         r.Limit,
		}
	}

	symbols := make([]SymbolInfo, len(resp.Symbols))
	for i, s := range resp.Symbols {
		filters := make([]SymbolFilter, len(s.Filters))
		for j, f := range s.Filters {
			filters[j] = SymbolFilter{
				FilterType:  f.FilterType,
				MinPrice:    f.MinPrice,
				MaxPrice:    f.MaxPrice,
				TickSize:    f.TickSize,
				MinQty:      f.MinQty,
				MaxQty:      f.MaxQty,
				StepSize:    f.StepSize,
				MinNotional: f.MinNotional,
			}
		}

		symbols[i] = SymbolInfo{
			Symbol:                 s.Symbol,
			Status:                 s.Status,
			BaseAsset:              s.BaseAsset,
			BaseAssetPrecision:     int(s.BaseAssetPrecision),
			QuoteAsset:             s.QuoteAsset,
			QuotePrecision:         int(s.QuotePrecision),
			QuoteAssetPrecision:    int(s.QuoteAssetPrecision),
			OrderTypes:             s.OrderTypes,
			IcebergAllowed:         s.IcebergAllowed,
			OcoAllowed:             s.OcoAllowed,
			IsSpotTradingAllowed:   s.IsSpotTradingAllowed,
			IsMarginTradingAllowed: s.IsMarginTradingAllowed,
			Filters:                filters,
			Permissions:            s.Permissions,
		}
	}

	return &ExchangeInfo{
		Timezone:   resp.Timezone,
		ServerTime: int64(resp.ServerTime),
		RateLimits: rateLimits,
		Symbols:    symbols,
	}, nil
}

func (c *client) GetOrderBook(ctx context.Context, symbol string, limit int) (*OrderBookResponse, error) {
	svc := c.sdkClient.NewOrderBookService().Symbol(symbol)
	if limit > 0 {
		svc.Limit(limit)
	}

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	bids := make([][]string, len(resp.Bids))
	for i, b := range resp.Bids {
		if len(b) >= 2 {
			bids[i] = []string{bigFloatToString(b[0]), bigFloatToString(b[1])}
		}
	}

	asks := make([][]string, len(resp.Asks))
	for i, a := range resp.Asks {
		if len(a) >= 2 {
			asks[i] = []string{bigFloatToString(a[0]), bigFloatToString(a[1])}
		}
	}

	return &OrderBookResponse{
		LastUpdateID: int64(resp.LastUpdateId),
		Bids:         bids,
		Asks:         asks,
	}, nil
}

func (c *client) GetTicker24hr(ctx context.Context, symbol string) (*TickerResponse, error) {
	resp, err := c.sdkClient.NewTicker24hrService().Symbol(symbol).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get 24hr ticker: %w", err)
	}

	// The response is an array, but for single symbol it's one element
	if len(resp) == 0 {
		return nil, fmt.Errorf("no ticker data returned")
	}

	t := resp[0]
	return &TickerResponse{
		Symbol:             t.Symbol,
		PriceChange:        t.PriceChange,
		PriceChangePercent: t.PriceChangePercent,
		WeightedAvgPrice:   t.WeightedAvgPrice,
		PrevClosePrice:     t.PrevClosePrice,
		LastPrice:          t.LastPrice,
		LastQty:            t.LastQty,
		BidPrice:           t.BidPrice,
		AskPrice:           t.AskPrice,
		OpenPrice:          t.OpenPrice,
		HighPrice:          t.HighPrice,
		LowPrice:           t.LowPrice,
		Volume:             t.Volume,
		QuoteVolume:        t.QuoteVolume,
		OpenTime:           int64(t.OpenTime),
		CloseTime:          int64(t.CloseTime),
		FirstID:            int64(t.FirstId),
		LastID:             int64(t.LastId),
		Count:              int64(t.Count),
	}, nil
}

func (c *client) GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]KlineEntry, error) {
	svc := c.sdkClient.NewKlinesService().
		Symbol(symbol).
		Interval(interval)
	if limit > 0 {
		svc.Limit(limit)
	}

	resp, err := svc.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// Convert SDK response to our KlineEntry format ([]interface{})
	klines := make([]KlineEntry, len(resp))
	for i, k := range resp {
		klines[i] = KlineEntry{
			k.OpenTime,
			k.Open,
			k.High,
			k.Low,
			k.Close,
			k.Volume,
			k.CloseTime,
			k.QuoteAssetVolume,
			k.NumberOfTrades,
			k.TakerBuyBaseAssetVolume,
			k.TakerBuyQuoteAssetVolume,
			"0", // ignore field
		}
	}

	return klines, nil
}

// bigFloatToString converts a *big.Float to string, handling nil case
func bigFloatToString(f *big.Float) string {
	if f == nil {
		return "0"
	}
	return f.Text('f', -1)
}
