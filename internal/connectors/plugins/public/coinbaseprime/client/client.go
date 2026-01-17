package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/balances"
	primeclient "github.com/coinbase-samples/prime-sdk-go/client"
	"github.com/coinbase-samples/prime-sdk-go/credentials"
	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/coinbase-samples/prime-sdk-go/orders"
	"github.com/coinbase-samples/prime-sdk-go/portfolios"
	"github.com/coinbase-samples/prime-sdk-go/wallets"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

const (
	defaultTimeout = 30 * time.Second
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	// Portfolio operations
	GetPortfolio(ctx context.Context) (*portfolios.GetPortfolioResponse, error)

	// Wallet/Account operations
	GetWallets(ctx context.Context, cursor string, limit int) (*wallets.ListWalletsResponse, error)

	// Balance operations
	GetPortfolioBalances(ctx context.Context) (*balances.ListPortfolioBalancesResponse, error)

	// Order operations
	ListOrders(ctx context.Context, params ListOrdersParams) (*orders.ListOrdersResponse, error)
	GetOrder(ctx context.Context, orderID string) (*orders.GetOrderResponse, error)
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*orders.CreateOrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*orders.CancelOrderResponse, error)

	// Conversion operations
	CreateConversion(ctx context.Context, req CreateConversionRequest) (*CreateConversionResponse, error)
	GetConversion(ctx context.Context, conversionID string) (*Conversion, error)

	// Market data operations
	GetOrderBook(ctx context.Context, productID string, depth int) (*OrderBookResponse, error)
	GetProducts(ctx context.Context) ([]Product, error)
}

type client struct {
	restClient primeclient.RestClient
	creds      *credentials.Credentials
	httpClient *http.Client

	portfolioService portfolios.PortfoliosService
	walletsService   wallets.WalletsService
	balancesService  balances.BalancesService
	ordersService    orders.OrdersService
}

func New(
	connectorName string,
	accessKey string,
	passphrase string,
	signingKey string,
	portfolioID string,
	svcAccountID string,
	entityID string,
) Client {
	creds := &credentials.Credentials{
		AccessKey:    accessKey,
		Passphrase:   passphrase,
		SigningKey:   signingKey,
		PortfolioId:  portfolioID,
		SvcAccountId: svcAccountID,
		EntityId:     entityID,
	}

	httpClient := metrics.NewHTTPClient(connectorName, defaultTimeout)
	restClient := primeclient.NewRestClient(creds, *httpClient)

	return &client{
		restClient:       restClient,
		creds:            creds,
		httpClient:       httpClient,
		portfolioService: portfolios.NewPortfoliosService(restClient),
		walletsService:   wallets.NewWalletsService(restClient),
		balancesService:  balances.NewBalancesService(restClient),
		ordersService:    orders.NewOrdersService(restClient),
	}
}

func (c *client) GetPortfolio(ctx context.Context) (*portfolios.GetPortfolioResponse, error) {
	return c.portfolioService.GetPortfolio(ctx, &portfolios.GetPortfolioRequest{
		PortfolioId: c.creds.PortfolioId,
	})
}

func (c *client) GetWallets(ctx context.Context, cursor string, limit int) (*wallets.ListWalletsResponse, error) {
	req := &wallets.ListWalletsRequest{
		PortfolioId: c.creds.PortfolioId,
	}

	return c.walletsService.ListWallets(ctx, req)
}

func (c *client) GetPortfolioBalances(ctx context.Context) (*balances.ListPortfolioBalancesResponse, error) {
	return c.balancesService.ListPortfolioBalances(ctx, &balances.ListPortfolioBalancesRequest{
		PortfolioId: c.creds.PortfolioId,
	})
}

func (c *client) ListOrders(ctx context.Context, params ListOrdersParams) (*orders.ListOrdersResponse, error) {
	req := &orders.ListOrdersRequest{
		PortfolioId: c.creds.PortfolioId,
		Start:       params.StartDate,
	}

	if len(params.OrderStatuses) > 0 {
		req.Statuses = params.OrderStatuses
	}
	if len(params.ProductIDs) > 0 {
		req.ProductIds = params.ProductIDs
	}
	if params.OrderType != "" {
		req.Type = params.OrderType
	}
	if params.OrderSide != "" {
		req.OrderSide = params.OrderSide
	}
	if !params.EndDate.IsZero() {
		req.End = params.EndDate
	}

	return c.ordersService.ListOrders(ctx, req)
}

func (c *client) GetOrder(ctx context.Context, orderID string) (*orders.GetOrderResponse, error) {
	return c.ordersService.GetOrder(ctx, &orders.GetOrderRequest{
		PortfolioId: c.creds.PortfolioId,
		OrderId:     orderID,
	})
}

func (c *client) CreateOrder(ctx context.Context, req CreateOrderRequest) (*orders.CreateOrderResponse, error) {
	createReq := &orders.CreateOrderRequest{
		Order: &model.Order{
			PortfolioId:   c.creds.PortfolioId,
			ProductId:     req.ProductID,
			Side:          req.Side,
			Type:          req.Type,
			BaseQuantity:  req.BaseQuantity,
			QuoteValue:    req.QuoteValue,
			LimitPrice:    req.LimitPrice,
			ClientOrderId: req.ClientOrderID,
			TimeInForce:   req.TimeInForce,
			ExpiryTime:    req.ExpiryTime,
		},
	}

	return c.ordersService.CreateOrder(ctx, createReq)
}

func (c *client) CancelOrder(ctx context.Context, orderID string) (*orders.CancelOrderResponse, error) {
	return c.ordersService.CancelOrder(ctx, &orders.CancelOrderRequest{
		PortfolioId: c.creds.PortfolioId,
		OrderId:     orderID,
	})
}

// CreateConversion is not directly supported by the SDK, uses custom implementation
func (c *client) CreateConversion(ctx context.Context, req CreateConversionRequest) (*CreateConversionResponse, error) {
	// Coinbase Prime conversions are handled differently than orders
	// This would need custom HTTP call or use transactions API
	// For now, return a placeholder
	return nil, nil
}

// GetConversion is not directly supported by the SDK
func (c *client) GetConversion(ctx context.Context, conversionID string) (*Conversion, error) {
	// Coinbase Prime conversions are tracked via transactions
	// This would need custom implementation
	return nil, nil
}

// GetOrderBook fetches the order book from Coinbase Exchange API (public endpoint)
// The productID should be in the format like "BTC-USD"
func (c *client) GetOrderBook(ctx context.Context, productID string, depth int) (*OrderBookResponse, error) {
	// Use the public Coinbase Exchange API for order book data
	// This is a public endpoint that doesn't require authentication
	url := fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/book?level=2", productID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// The Coinbase response format is slightly different, parse it
	var rawResp struct {
		Sequence int64      `json:"sequence"`
		Bids     [][]string `json:"bids"` // [price, size, num_orders]
		Asks     [][]string `json:"asks"` // [price, size, num_orders]
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our format
	orderBook := &OrderBookResponse{
		ProductID: productID,
		Sequence:  rawResp.Sequence,
		Time:      time.Now().UTC(),
		Bids:      make([]OrderBookEntry, 0, len(rawResp.Bids)),
		Asks:      make([]OrderBookEntry, 0, len(rawResp.Asks)),
	}

	// Apply depth limit
	bidLimit := len(rawResp.Bids)
	askLimit := len(rawResp.Asks)
	if depth > 0 {
		if depth < bidLimit {
			bidLimit = depth
		}
		if depth < askLimit {
			askLimit = depth
		}
	}

	for i := 0; i < bidLimit; i++ {
		bid := rawResp.Bids[i]
		if len(bid) >= 2 {
			orderBook.Bids = append(orderBook.Bids, OrderBookEntry{
				Price: bid[0],
				Size:  bid[1],
			})
		}
	}

	for i := 0; i < askLimit; i++ {
		ask := rawResp.Asks[i]
		if len(ask) >= 2 {
			orderBook.Asks = append(orderBook.Asks, OrderBookEntry{
				Price: ask[0],
				Size:  ask[1],
			})
		}
	}

	return orderBook, nil
}

// GetProducts fetches all tradable products from Coinbase Exchange API (public endpoint)
func (c *client) GetProducts(ctx context.Context) ([]Product, error) {
	url := "https://api.exchange.coinbase.com/products"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var products []Product
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return products, nil
}

