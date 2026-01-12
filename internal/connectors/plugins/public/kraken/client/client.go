package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/krakenfx/api-go/v2/pkg/spot"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	// Balance operations
	GetBalance(ctx context.Context) (map[string]string, error)

	// Order operations
	GetClosedOrders(ctx context.Context, params ListOrdersParams) (*ClosedOrdersResponse, error)
	GetOpenOrders(ctx context.Context, params ListOrdersParams) (*OpenOrdersResponse, error)
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error)

	// Asset operations
	GetAssetPairs(ctx context.Context) (map[string]AssetPair, error)

	// Market data operations
	GetOrderBook(ctx context.Context, pair string, depth int) (*OrderBookResponse, error)
	GetTicker(ctx context.Context, pair string) (*TickerResponse, error)
	GetOHLC(ctx context.Context, pair string, interval int, since *int64) (*OHLCResponse, error)
}

type client struct {
	rest *spot.REST
}

func New(
	connectorName string,
	endpoint string,
	publicKey string,
	privateKey string,
) Client {
	// Ensure endpoint doesn't have trailing slash
	endpoint = strings.TrimSuffix(endpoint, "/")

	rest := spot.NewREST()
	rest.PublicKey = publicKey
	rest.PrivateKey = privateKey
	rest.BaseURL = endpoint

	return &client{
		rest: rest,
	}
}

func (c *client) GetBalance(ctx context.Context) (map[string]string, error) {
	resp, err := c.rest.Balances()
	if err != nil {
		return nil, fmt.Errorf("failed to get balances: %w", err)
	}

	result := make(map[string]string)
	for asset, amount := range resp.Result {
		if amount != nil {
			result[asset] = amount.String()
		}
	}

	return result, nil
}

func (c *client) GetClosedOrders(ctx context.Context, params ListOrdersParams) (*ClosedOrdersResponse, error) {
	req := &spot.ClosedOrdersRequest{
		Trades:  params.Trades,
		Userref: params.UserRef,
	}

	if !params.Start.IsZero() {
		req.Start = int(params.Start.Unix())
	}
	if !params.End.IsZero() {
		req.End = int(params.End.Unix())
	}
	if params.Offset > 0 {
		req.Ofs = params.Offset
	}
	if params.CloseTime != "" {
		req.CloseTime = params.CloseTime
	}

	resp, err := c.rest.ClosedOrders(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get closed orders: %w", err)
	}

	orders := make(map[string]Order)
	for txid, order := range resp.Result.Closed {
		orders[txid] = convertSDKOrderToOrder(order)
	}

	return &ClosedOrdersResponse{
		Orders: orders,
	}, nil
}

func (c *client) GetOpenOrders(ctx context.Context, params ListOrdersParams) (*OpenOrdersResponse, error) {
	req := &spot.OpenOrdersRequest{
		Trades:  params.Trades,
		Userref: params.UserRef,
	}

	resp, err := c.rest.OpenOrders(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	orders := make(map[string]Order)
	for txid, order := range resp.Result.Open {
		orders[txid] = convertSDKOpenOrderToOrder(order)
	}

	return &OpenOrdersResponse{
		Orders: orders,
	}, nil
}

func (c *client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error) {
	sdkReq := &spot.AddOrderRequest{
		OrderType:   req.OrderType,
		Type:        req.Type,
		Volume:      req.Volume,
		Pair:        req.Pair,
		Price:       req.Price,
		TimeInForce: req.TimeInForce,
		ClOrdId:     req.ClientOrderID,
		Validate:    req.Validate,
	}

	if req.Price2 != "" {
		sdkReq.SecondaryPrice = req.Price2
	}
	if req.Trigger != "" {
		sdkReq.Trigger = req.Trigger
	}
	if req.Leverage != "" {
		sdkReq.Leverage = req.Leverage
	}
	if req.ReduceOnly {
		sdkReq.ReduceOnly = true
	}
	if req.StartTm != "" {
		sdkReq.StartTm = req.StartTm
	}
	if req.ExpireTm != "" {
		sdkReq.ExpireTm = req.ExpireTm
	}

	resp, err := c.rest.AddOrder(sdkReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	var txids []string
	if resp.Result.ID != nil {
		txids = append(txids, resp.Result.ID...)
	}

	return &CreateOrderResponse{
		TxID: txids,
	}, nil
}

func (c *client) CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	resp, err := c.rest.CancelOrder(&spot.CancelOrderRequest{
		TxID: orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return &CancelOrderResponse{
		Count:   resp.Result.Count,
		Pending: resp.Result.Pending,
	}, nil
}

func (c *client) GetAssetPairs(ctx context.Context) (map[string]AssetPair, error) {
	resp, err := c.rest.AssetPairs(&spot.AssetPairsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get asset pairs: %w", err)
	}

	result := make(map[string]AssetPair)
	for name, pair := range resp.Result {
		ap := AssetPair{
			Altname:       pair.AltName,
			WSName:        pair.WSName,
			AClassBase:    pair.BaseAssetClass,
			Base:          pair.Base,
			AClassQuote:   pair.QuoteAssetClass,
			Quote:         pair.Quote,
			CostDecimals:  pair.CostDecimals,
			PairDecimals:  pair.PairDecimals,
			LotDecimals:   pair.LotDecimals,
			LotMultiplier: pair.LotMultiplier,
			Status:        pair.Status,
		}
		if pair.OrderMinimum != nil {
			ap.OrderMin = pair.OrderMinimum.String()
		}
		if pair.CostMinimum != nil {
			ap.CostMin = pair.CostMinimum.String()
		}
		if pair.TickSize != nil {
			ap.TickSize = pair.TickSize.String()
		}
		result[name] = ap
	}

	return result, nil
}

func (c *client) GetOrderBook(ctx context.Context, pair string, depth int) (*OrderBookResponse, error) {
	req := &spot.OrderBookRequest{
		Pair: pair,
	}
	if depth > 0 {
		req.Count = depth
	}

	resp, err := c.rest.OrderBook(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	// Find the order book data (there's only one key, the pair name)
	var book *spot.OrderBook
	for _, v := range resp.Result {
		book = &v
		break
	}

	if book == nil {
		return nil, fmt.Errorf("no order book data found for pair %s", pair)
	}

	result := &OrderBookResponse{
		Asks: make([]OrderBookEntry, 0, len(book.Asks)),
		Bids: make([]OrderBookEntry, 0, len(book.Bids)),
	}

	for _, ask := range book.Asks {
		result.Asks = append(result.Asks, OrderBookEntry{
			Price:     ask.Price.String(),
			Volume:    ask.Volume.String(),
			Timestamp: ask.Timestamp.Unix(),
		})
	}

	for _, bid := range book.Bids {
		result.Bids = append(result.Bids, OrderBookEntry{
			Price:     bid.Price.String(),
			Volume:    bid.Volume.String(),
			Timestamp: bid.Timestamp.Unix(),
		})
	}

	return result, nil
}

func (c *client) GetTicker(ctx context.Context, pair string) (*TickerResponse, error) {
	resp, err := c.rest.Ticker(&spot.TickerRequest{
		Pair: pair,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker: %w", err)
	}

	// Find the ticker data (there's only one key, the pair name)
	var tickerInfo *spot.AssetTickerInfo
	for _, v := range resp.Result {
		tickerInfo = &v
		break
	}

	if tickerInfo == nil {
		return nil, fmt.Errorf("no ticker data found for pair %s", pair)
	}

	// Convert SDK ticker to our format
	tickerData := TickerData{}

	if tickerInfo.Open != nil {
		tickerData.O = tickerInfo.Open.String()
	}

	// Ask prices
	if len(tickerInfo.Ask) >= 3 {
		tickerData.A = []string{
			tickerInfo.Ask[0].String(),
			tickerInfo.Ask[1].String(),
			tickerInfo.Ask[2].String(),
		}
	}

	// Bid prices
	if len(tickerInfo.Bid) >= 3 {
		tickerData.B = []string{
			tickerInfo.Bid[0].String(),
			tickerInfo.Bid[1].String(),
			tickerInfo.Bid[2].String(),
		}
	}

	// Last trade
	if len(tickerInfo.Close) >= 2 {
		tickerData.C = []string{
			tickerInfo.Close[0].String(),
			tickerInfo.Close[1].String(),
		}
	}

	// Volume
	if len(tickerInfo.Volume) >= 2 {
		tickerData.V = []string{
			tickerInfo.Volume[0].String(),
			tickerInfo.Volume[1].String(),
		}
	}

	// VWAP
	if len(tickerInfo.VWAP) >= 2 {
		tickerData.P = []string{
			tickerInfo.VWAP[0].String(),
			tickerInfo.VWAP[1].String(),
		}
	}

	// Number of trades
	if len(tickerInfo.Trades) >= 2 {
		tickerData.T = []int{tickerInfo.Trades[0], tickerInfo.Trades[1]}
	}

	// Low
	if len(tickerInfo.Low) >= 2 {
		tickerData.L = []string{
			tickerInfo.Low[0].String(),
			tickerInfo.Low[1].String(),
		}
	}

	// High
	if len(tickerInfo.High) >= 2 {
		tickerData.H = []string{
			tickerInfo.High[0].String(),
			tickerInfo.High[1].String(),
		}
	}

	return &TickerResponse{
		Data: tickerData,
	}, nil
}

func (c *client) GetOHLC(ctx context.Context, pair string, interval int, since *int64) (*OHLCResponse, error) {
	req := &spot.OHLCRequest{
		Pair:     pair,
		Interval: interval,
	}
	if since != nil {
		req.Since = int(*since)
	}

	resp, err := c.rest.OHLC(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get OHLC: %w", err)
	}

	// Extract the OHLC data
	var entries []OHLCEntryData
	var last int64

	for key, value := range resp.Result {
		if key == "last" {
			if lastNum, ok := value.(json.Number); ok {
				last, _ = lastNum.Int64()
			} else if lastFloat, ok := value.(float64); ok {
				last = int64(lastFloat)
			}
			continue
		}

		// This is the OHLC data array
		if candles, ok := value.([]interface{}); ok {
			for _, candle := range candles {
				if candleArr, ok := candle.([]interface{}); ok && len(candleArr) >= 7 {
					entry := OHLCEntryData{}

					if ts, ok := candleArr[0].(float64); ok {
						entry.Timestamp = int64(ts)
					} else if tsNum, ok := candleArr[0].(json.Number); ok {
						entry.Timestamp, _ = tsNum.Int64()
					}

					entry.Open = interfaceToString(candleArr[1])
					entry.High = interfaceToString(candleArr[2])
					entry.Low = interfaceToString(candleArr[3])
					entry.Close = interfaceToString(candleArr[4])
					entry.VWAP = interfaceToString(candleArr[5])
					entry.Volume = interfaceToString(candleArr[6])

					if len(candleArr) > 7 {
						if count, ok := candleArr[7].(float64); ok {
							entry.Count = int(count)
						} else if countNum, ok := candleArr[7].(json.Number); ok {
							countInt, _ := countNum.Int64()
							entry.Count = int(countInt)
						}
					}

					entries = append(entries, entry)
				}
			}
		}
	}

	return &OHLCResponse{
		Entries: entries,
		Last:    last,
	}, nil
}

func interfaceToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case json.Number:
		return val.String()
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// convertSDKOrderToOrder converts a spot.ClosedOrder to our Order type
func convertSDKOrderToOrder(sdkOrder spot.ClosedOrder) Order {
	var openTime, closeTime, startTime, expireTime float64
	var userRef int

	if sdkOrder.OpenTm != nil {
		openTime = sdkOrder.OpenTm.Float64()
	}
	if sdkOrder.CloseTm != nil {
		closeTime = sdkOrder.CloseTm.Float64()
	}
	if sdkOrder.StartTm != nil {
		startTime = sdkOrder.StartTm.Float64()
	}
	if sdkOrder.ExpireTm != nil {
		expireTime = sdkOrder.ExpireTm.Float64()
	}
	if sdkOrder.UserRef != nil {
		userRef = int(sdkOrder.UserRef.Int64())
	}

	order := Order{
		RefID:         sdkOrder.RefID,
		UserRef:       userRef,
		Status:        sdkOrder.Status,
		OpenTime:      openTime,
		CloseTime:     closeTime,
		StartTime:     startTime,
		ExpireTime:    expireTime,
		Vol:           safeDecimalString(sdkOrder.Volume),
		VolExec:       safeDecimalString(sdkOrder.VolumeExecuted),
		Cost:          safeDecimalString(sdkOrder.Cost),
		Fee:           safeDecimalString(sdkOrder.Fee),
		Price:         safeDecimalString(sdkOrder.Price),
		StopPrice:     safeDecimalString(sdkOrder.StopPrice),
		LimitPrice:    safeDecimalString(sdkOrder.LimitPrice),
		Trigger:       sdkOrder.Trigger,
		Misc:          sdkOrder.Misc,
		OFlags:        sdkOrder.OrderFlags,
		ClientOrderID: sdkOrder.ClOrdID,
	}

	if sdkOrder.Description != nil {
		order.Descr = OrderDesc{
			Pair:      sdkOrder.Description.Pair,
			Type:      sdkOrder.Description.Type,
			OrderType: sdkOrder.Description.OrderType,
			Price:     safeDecimalString(sdkOrder.Description.Price),
			Price2:    safeDecimalString(sdkOrder.Description.SecondaryPrice),
			Leverage:  sdkOrder.Description.Leverage,
			Order:     sdkOrder.Description.Order,
			Close:     sdkOrder.Description.Close,
		}
	}

	return order
}

// convertSDKOpenOrderToOrder converts a spot.Order to our Order type
func convertSDKOpenOrderToOrder(sdkOrder spot.Order) Order {
	var openTime, startTime, expireTime float64
	var userRef int

	if sdkOrder.OpenTm != nil {
		openTime = sdkOrder.OpenTm.Float64()
	}
	if sdkOrder.StartTm != nil {
		startTime = sdkOrder.StartTm.Float64()
	}
	if sdkOrder.ExpireTm != nil {
		expireTime = sdkOrder.ExpireTm.Float64()
	}
	if sdkOrder.UserRef != nil {
		userRef = int(sdkOrder.UserRef.Int64())
	}

	order := Order{
		RefID:         sdkOrder.RefID,
		UserRef:       userRef,
		Status:        sdkOrder.Status,
		OpenTime:      openTime,
		StartTime:     startTime,
		ExpireTime:    expireTime,
		Vol:           safeDecimalString(sdkOrder.Volume),
		VolExec:       safeDecimalString(sdkOrder.VolumeExecuted),
		Cost:          safeDecimalString(sdkOrder.Cost),
		Fee:           safeDecimalString(sdkOrder.Fee),
		Price:         safeDecimalString(sdkOrder.Price),
		StopPrice:     safeDecimalString(sdkOrder.StopPrice),
		LimitPrice:    safeDecimalString(sdkOrder.LimitPrice),
		Trigger:       sdkOrder.Trigger,
		Misc:          sdkOrder.Misc,
		OFlags:        sdkOrder.OrderFlags,
		ClientOrderID: sdkOrder.ClOrdID,
	}

	if sdkOrder.Description != nil {
		order.Descr = OrderDesc{
			Pair:      sdkOrder.Description.Pair,
			Type:      sdkOrder.Description.Type,
			OrderType: sdkOrder.Description.OrderType,
			Price:     safeDecimalString(sdkOrder.Description.Price),
			Price2:    safeDecimalString(sdkOrder.Description.SecondaryPrice),
			Leverage:  sdkOrder.Description.Leverage,
			Order:     sdkOrder.Description.Order,
			Close:     sdkOrder.Description.Close,
		}
	}

	return order
}

func safeDecimalString(d interface{}) string {
	if d == nil {
		return ""
	}
	if stringer, ok := d.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%v", d)
}
