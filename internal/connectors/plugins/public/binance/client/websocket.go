package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsBaseURL        = "wss://stream.binance.com:9443/ws"
	wsTestnetBaseURL = "wss://testnet.binance.vision/ws"

	// Binance requires keepalive pings every 30 minutes for user data streams
	listenKeyRefreshInterval = 30 * time.Minute
	// Websocket ping interval
	wsPingInterval = 3 * time.Minute
)

// OrderUpdateEvent represents an order update from Binance WebSocket
type OrderUpdateEvent struct {
	EventType              string `json:"e"` // "executionReport"
	EventTime              int64  `json:"E"`
	Symbol                 string `json:"s"`
	ClientOrderID          string `json:"c"`
	Side                   string `json:"S"` // BUY or SELL
	OrderType              string `json:"o"` // LIMIT, MARKET, etc.
	TimeInForce            string `json:"f"` // GTC, IOC, FOK
	Quantity               string `json:"q"` // Original quantity
	Price                  string `json:"p"` // Order price
	StopPrice              string `json:"P"` // Stop price
	ExecutionType          string `json:"x"` // NEW, TRADE, CANCELED, etc.
	OrderStatus            string `json:"X"` // NEW, PARTIALLY_FILLED, FILLED, CANCELED, etc.
	OrderRejectReason      string `json:"r"`
	OrderID                int64  `json:"i"`
	LastExecutedQuantity   string `json:"l"`
	CumulativeFilledQty    string `json:"z"`
	LastExecutedPrice      string `json:"L"`
	CommissionAmount       string `json:"n"`
	CommissionAsset        string `json:"N"`
	TransactionTime        int64  `json:"T"`
	TradeID                int64  `json:"t"`
	IsOnOrderBook          bool   `json:"w"`
	IsMakerTrade           bool   `json:"m"`
	OrderCreationTime      int64  `json:"O"`
	CumulativeQuoteQty     string `json:"Z"`
	LastQuoteQty           string `json:"Y"`
	QuoteOrderQty          string `json:"Q"`
}

// WebSocketMessage represents a generic WebSocket message from Binance
type WebSocketMessage struct {
	EventType string          `json:"e"`
	RawData   json.RawMessage `json:"-"`
}

// OrderUpdateCallback is called when an order update is received
type OrderUpdateCallback func(event OrderUpdateEvent)

// WebSocketClient manages WebSocket connections to Binance
type WebSocketClient struct {
	httpClient      *http.Client
	baseURL         string
	wsBaseURL       string
	apiKey          string
	secretKey       string
	listenKey       string
	conn            *websocket.Conn
	mu              sync.Mutex
	stopCh          chan struct{}
	callback        OrderUpdateCallback
	autoReconnect   bool
	reconnectDelay  time.Duration
	maxReconnectDelay time.Duration
}

// NewWebSocketClient creates a new WebSocket client for Binance
func NewWebSocketClient(
	httpClient *http.Client,
	baseURL string,
	apiKey string,
	secretKey string,
	testNet bool,
) *WebSocketClient {
	wsURL := wsBaseURL
	if testNet {
		wsURL = wsTestnetBaseURL
	}

	return &WebSocketClient{
		httpClient:        httpClient,
		baseURL:           baseURL,
		wsBaseURL:         wsURL,
		apiKey:            apiKey,
		secretKey:         secretKey,
		autoReconnect:     true,
		reconnectDelay:    time.Second,
		maxReconnectDelay: time.Minute,
	}
}

// createListenKey requests a listen key from Binance for user data stream
func (c *WebSocketClient) createListenKey(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v3/userDataStream", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		ListenKey string `json:"listenKey"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ListenKey, nil
}

// keepAliveListenKey sends a keepalive ping for the listen key
func (c *WebSocketClient) keepAliveListenKey(ctx context.Context) error {
	if c.listenKey == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		c.baseURL+"/api/v3/userDataStream?listenKey="+c.listenKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// closeListenKey closes the listen key
func (c *WebSocketClient) closeListenKey(ctx context.Context) error {
	if c.listenKey == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+"/api/v3/userDataStream?listenKey="+c.listenKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// Start connects to the Binance WebSocket and starts receiving order updates
func (c *WebSocketClient) Start(ctx context.Context, callback OrderUpdateCallback) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return fmt.Errorf("WebSocket connection already active")
	}

	c.callback = callback
	c.stopCh = make(chan struct{})

	// Get listen key for user data stream
	listenKey, err := c.createListenKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to create listen key: %w", err)
	}
	c.listenKey = listenKey

	// Connect to WebSocket
	if err := c.connect(ctx); err != nil {
		return err
	}

	// Start background goroutines
	go c.readLoop()
	go c.keepAliveLoop(ctx)

	return nil
}

// connect establishes the WebSocket connection
func (c *WebSocketClient) connect(ctx context.Context) error {
	wsURL := fmt.Sprintf("%s/%s", c.wsBaseURL, c.listenKey)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	return nil
}

// readLoop reads messages from the WebSocket
func (c *WebSocketClient) readLoop() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}

			// Attempt reconnection if enabled
			if c.autoReconnect {
				c.handleReconnect()
				continue
			}
			return
		}

		// Parse the message
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		// Handle order updates (execution reports)
		if wsMsg.EventType == "executionReport" {
			var event OrderUpdateEvent
			if err := json.Unmarshal(message, &event); err != nil {
				continue
			}

			if c.callback != nil {
				c.callback(event)
			}
		}
	}
}

// handleReconnect attempts to reconnect with exponential backoff
func (c *WebSocketClient) handleReconnect() {
	delay := c.reconnectDelay

	for {
		select {
		case <-c.stopCh:
			return
		case <-time.After(delay):
		}

		c.mu.Lock()
		// Get new listen key
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		listenKey, err := c.createListenKey(ctx)
		cancel()

		if err == nil {
			c.listenKey = listenKey
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err = c.connect(ctx)
			cancel()
		}
		c.mu.Unlock()

		if err == nil {
			// Reset delay on successful reconnect
			c.reconnectDelay = time.Second
			return
		}

		// Exponential backoff
		delay *= 2
		if delay > c.maxReconnectDelay {
			delay = c.maxReconnectDelay
		}
	}
}

// keepAliveLoop sends periodic keepalive pings for the listen key
func (c *WebSocketClient) keepAliveLoop(ctx context.Context) {
	ticker := time.NewTicker(listenKeyRefreshInterval)
	defer ticker.Stop()

	pingTicker := time.NewTicker(wsPingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Refresh listen key
			if err := c.keepAliveListenKey(ctx); err != nil {
				// If keepalive fails, try to get a new listen key
				newKey, err := c.createListenKey(ctx)
				if err == nil {
					c.mu.Lock()
					c.listenKey = newKey
					c.mu.Unlock()
				}
			}
		case <-pingTicker.C:
			// Send WebSocket ping
			c.mu.Lock()
			if c.conn != nil {
				_ = c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
			}
			c.mu.Unlock()
		}
	}
}

// Stop closes the WebSocket connection and cleans up resources
func (c *WebSocketClient) Stop(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Signal stop to all goroutines
	if c.stopCh != nil {
		close(c.stopCh)
	}

	// Close WebSocket connection
	if c.conn != nil {
		_ = c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
		c.conn = nil
	}

	// Close listen key
	_ = c.closeListenKey(ctx)
	c.listenKey = ""
}

// IsConnected returns whether the WebSocket is currently connected
func (c *WebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// SetAutoReconnect enables or disables automatic reconnection
func (c *WebSocketClient) SetAutoReconnect(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.autoReconnect = enabled
}
