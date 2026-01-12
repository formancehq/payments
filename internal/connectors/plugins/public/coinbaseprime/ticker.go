package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTicker(ctx context.Context, req models.GetTickerRequest) (models.GetTickerResponse, error) {
	// Convert pair format from "BTC/USD" to "BTC-USD" for Coinbase
	productID := strings.ReplaceAll(req.Pair, "/", "-")

	ticker, err := p.fetchTickerFromExchange(ctx, productID)
	if err != nil {
		return models.GetTickerResponse{}, fmt.Errorf("failed to get ticker: %w", err)
	}

	ticker.Pair = req.Pair
	return models.GetTickerResponse{
		Ticker: ticker,
	}, nil
}

// fetchTickerFromExchange fetches ticker data from Coinbase Exchange public API
func (p *Plugin) fetchTickerFromExchange(ctx context.Context, productID string) (models.Ticker, error) {
	// Use the public Coinbase Exchange API for ticker data
	// We'll fetch both ticker and 24hr stats
	tickerURL := fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/ticker", productID)
	statsURL := fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/stats", productID)

	// Fetch ticker
	tickerReq, err := http.NewRequestWithContext(ctx, "GET", tickerURL, nil)
	if err != nil {
		return models.Ticker{}, fmt.Errorf("failed to create ticker request: %w", err)
	}
	tickerReq.Header.Set("Accept", "application/json")

	tickerResp, err := http.DefaultClient.Do(tickerReq)
	if err != nil {
		return models.Ticker{}, fmt.Errorf("failed to execute ticker request: %w", err)
	}
	defer tickerResp.Body.Close()

	if tickerResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tickerResp.Body)
		return models.Ticker{}, fmt.Errorf("unexpected ticker status code %d: %s", tickerResp.StatusCode, string(body))
	}

	var tickerData struct {
		TradeID int64  `json:"trade_id"`
		Price   string `json:"price"`
		Size    string `json:"size"`
		Bid     string `json:"bid"`
		Ask     string `json:"ask"`
		Volume  string `json:"volume"`
		Time    string `json:"time"`
	}

	if err := json.NewDecoder(tickerResp.Body).Decode(&tickerData); err != nil {
		return models.Ticker{}, fmt.Errorf("failed to decode ticker response: %w", err)
	}

	// Fetch 24hr stats
	statsReq, err := http.NewRequestWithContext(ctx, "GET", statsURL, nil)
	if err != nil {
		return models.Ticker{}, fmt.Errorf("failed to create stats request: %w", err)
	}
	statsReq.Header.Set("Accept", "application/json")

	statsResp, err := http.DefaultClient.Do(statsReq)
	if err != nil {
		return models.Ticker{}, fmt.Errorf("failed to execute stats request: %w", err)
	}
	defer statsResp.Body.Close()

	var statsData struct {
		Open        string `json:"open"`
		High        string `json:"high"`
		Low         string `json:"low"`
		Last        string `json:"last"`
		Volume      string `json:"volume"`
		Volume30Day string `json:"volume_30day"`
	}

	if statsResp.StatusCode == http.StatusOK {
		// Ignore error - stats data is optional
		_ = json.NewDecoder(statsResp.Body).Decode(&statsData)
	}

	// Parse all values
	lastPrice, _ := parseDecimalString(tickerData.Price, 8)
	bidPrice, _ := parseDecimalString(tickerData.Bid, 8)
	askPrice, _ := parseDecimalString(tickerData.Ask, 8)
	volume24h, _ := parseDecimalString(tickerData.Volume, 8)
	high24h, _ := parseDecimalString(statsData.High, 8)
	low24h, _ := parseDecimalString(statsData.Low, 8)
	openPrice, _ := parseDecimalString(statsData.Open, 8)

	// Calculate price change
	var priceChange *big.Int
	if openPrice != nil && lastPrice != nil && openPrice.Cmp(big.NewInt(0)) != 0 {
		priceChange = new(big.Int).Sub(lastPrice, openPrice)
	}

	return models.Ticker{
		LastPrice:   lastPrice,
		BidPrice:    bidPrice,
		AskPrice:    askPrice,
		Volume24h:   volume24h,
		High24h:     high24h,
		Low24h:      low24h,
		OpenPrice:   openPrice,
		PriceChange: priceChange,
		Timestamp:   time.Now().UTC(),
	}, nil
}
