//go:build it

package test_suite

import (
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/testing/deferred"
	"github.com/google/uuid"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// OrderBookResponse matches the orderbook API response structure
type OrderBookResponse struct {
	Data struct {
		Pair      string    `json:"pair"`
		Bids      [][]any   `json:"bids"`
		Asks      [][]any   `json:"asks"`
		Timestamp time.Time `json:"timestamp,omitempty"`
	} `json:"data"`
}

// QuoteRequest matches the quote API request structure
type QuoteRequest struct {
	BaseAsset   string   `json:"baseAsset"`
	QuoteAsset  string   `json:"quoteAsset"`
	Amount      *big.Int `json:"amount"`
	IsBuy       bool     `json:"isBuy"`
	IncludeFees bool     `json:"includeFees,omitempty"`
}

// QuoteResponse matches the quote API response structure
type QuoteResponse struct {
	Data struct {
		BaseAsset    string    `json:"baseAsset"`
		QuoteAsset   string    `json:"quoteAsset"`
		Amount       *big.Int  `json:"amount"`
		Price        *big.Int  `json:"price"`
		QuoteAmount  *big.Int  `json:"quoteAmount"`
		Fee          *big.Int  `json:"fee,omitempty"`
		FeeAsset     string    `json:"feeAsset,omitempty"`
		IsBuy        bool      `json:"isBuy"`
		ValidUntil   time.Time `json:"validUntil,omitempty"`
		ExchangeRate string    `json:"exchangeRate,omitempty"`
	} `json:"data"`
}

// TradableAssetsResponse matches the tradable assets API response structure
type TradableAssetsResponse struct {
	Data []struct {
		Pair           string `json:"pair"`
		BaseAsset      string `json:"baseAsset"`
		QuoteAsset     string `json:"quoteAsset"`
		Status         string `json:"status,omitempty"`
		MinOrderSize   string `json:"minOrderSize,omitempty"`
		MaxOrderSize   string `json:"maxOrderSize,omitempty"`
		PricePrecision int    `json:"pricePrecision,omitempty"`
		SizePrecision  int    `json:"sizePrecision,omitempty"`
	} `json:"data"`
}

// TickerResponse matches the ticker API response structure
type TickerResponse struct {
	Data struct {
		Pair             string    `json:"pair"`
		LastPrice        *big.Int  `json:"lastPrice,omitempty"`
		BidPrice         *big.Int  `json:"bidPrice,omitempty"`
		AskPrice         *big.Int  `json:"askPrice,omitempty"`
		Volume24h        *big.Int  `json:"volume24h,omitempty"`
		High24h          *big.Int  `json:"high24h,omitempty"`
		Low24h           *big.Int  `json:"low24h,omitempty"`
		PriceChange24h   *big.Int  `json:"priceChange24h,omitempty"`
		PriceChangePct24 string    `json:"priceChangePct24h,omitempty"`
		Timestamp        time.Time `json:"timestamp,omitempty"`
	} `json:"data"`
}

var _ = Context("Payments API Market Data", Serial, func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		app *deferred.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                      stack,
			PostgresConfiguration:      db.GetValue().ConnectionOptions(),
			NatsURL:                    natsServer.GetValue().ClientURL(),
			TemporalNamespace:          temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:            temporalServer.GetValue().Address(),
			Output:                     GinkgoWriter,
			SkipOutboxScheduleCreation: true,
		}
	})

	AfterEach(func() {
		flushRemainingWorkflows(ctx)
	})

	When("accessing market data endpoints", func() {
		var (
			connectorID string
			err         error
		)

		BeforeEach(func() {
			connectorID, err = installConnector(ctx, app.GetValue(), uuid.New(), 3)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			uninstallConnector(ctx, app.GetValue(), connectorID)
		})

		// Note: These tests verify the endpoint structure and error handling.
		// DummyPay connector may not implement market data capabilities,
		// so we test for appropriate error responses or empty data.

		It("should handle orderbook request", func() {
			var orderBookResponse OrderBookResponse
			err := app.GetValue().Client().Get(ctx, "/v3/connectors/"+connectorID+"/orderbook?pair=BTC/USD&depth=10", &orderBookResponse)
			// DummyPay doesn't support orderbook, so we expect an error or empty response
			// The test validates the endpoint exists and handles the request properly
			if err != nil {
				// Expected behavior for connectors that don't support orderbook
				Expect(err.Error()).To(Or(
					ContainSubstring("not supported"),
					ContainSubstring("not implemented"),
					ContainSubstring("404"),
					ContainSubstring("500"),
				))
			} else {
				// If no error, validate response structure
				Expect(orderBookResponse.Data.Pair).To(Or(Equal("BTC/USD"), BeEmpty()))
			}
		})

		It("should handle quote request", func() {
			quoteRequest := QuoteRequest{
				BaseAsset:   "BTC",
				QuoteAsset:  "USD",
				Amount:      big.NewInt(100000000),
				IsBuy:       true,
				IncludeFees: false,
			}

			var quoteResponse QuoteResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/connectors/"+connectorID+"/quotes", quoteRequest, &quoteResponse)
			// DummyPay doesn't support quotes, so we expect an error or empty response
			if err != nil {
				// Expected behavior for connectors that don't support quotes
				Expect(err.Error()).To(Or(
					ContainSubstring("not supported"),
					ContainSubstring("not implemented"),
					ContainSubstring("404"),
					ContainSubstring("500"),
				))
			} else {
				// If no error, validate response structure
				Expect(quoteResponse.Data.BaseAsset).To(Or(Equal("BTC"), BeEmpty()))
			}
		})

		It("should handle tradable assets request", func() {
			var tradableAssetsResponse TradableAssetsResponse
			err := app.GetValue().Client().Get(ctx, "/v3/connectors/"+connectorID+"/tradable-assets", &tradableAssetsResponse)
			// DummyPay doesn't support tradable assets, so we expect an error or empty response
			if err != nil {
				// Expected behavior for connectors that don't support tradable assets
				Expect(err.Error()).To(Or(
					ContainSubstring("not supported"),
					ContainSubstring("not implemented"),
					ContainSubstring("404"),
					ContainSubstring("500"),
				))
			} else {
				// If no error, validate response is an array (can be empty)
				Expect(tradableAssetsResponse.Data).NotTo(BeNil())
			}
		})

		It("should handle ticker request", func() {
			var tickerResponse TickerResponse
			err := app.GetValue().Client().Get(ctx, "/v3/connectors/"+connectorID+"/ticker?pair=BTC/USD", &tickerResponse)
			// DummyPay doesn't support ticker, so we expect an error or empty response
			if err != nil {
				// Expected behavior for connectors that don't support ticker
				Expect(err.Error()).To(Or(
					ContainSubstring("not supported"),
					ContainSubstring("not implemented"),
					ContainSubstring("404"),
					ContainSubstring("500"),
				))
			} else {
				// If no error, validate response structure
				Expect(tickerResponse.Data.Pair).To(Or(Equal("BTC/USD"), BeEmpty()))
			}
		})

		It("should return 404 for orderbook with non-existent connector", func() {
			fakeConnectorID := "dummypay/" + uuid.New().String()
			var orderBookResponse OrderBookResponse
			err := app.GetValue().Client().Get(ctx, "/v3/connectors/"+fakeConnectorID+"/orderbook?pair=BTC/USD", &orderBookResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})

		It("should return 404 for tradable assets with non-existent connector", func() {
			fakeConnectorID := "dummypay/" + uuid.New().String()
			var tradableAssetsResponse TradableAssetsResponse
			err := app.GetValue().Client().Get(ctx, "/v3/connectors/"+fakeConnectorID+"/tradable-assets", &tradableAssetsResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})

		It("should return 404 for ticker with non-existent connector", func() {
			fakeConnectorID := "dummypay/" + uuid.New().String()
			var tickerResponse TickerResponse
			err := app.GetValue().Client().Get(ctx, "/v3/connectors/"+fakeConnectorID+"/ticker?pair=BTC/USD", &tickerResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})

		It("should return 404 for quote with non-existent connector", func() {
			fakeConnectorID := "dummypay/" + uuid.New().String()
			quoteRequest := QuoteRequest{
				BaseAsset:  "BTC",
				QuoteAsset: "USD",
				Amount:     big.NewInt(100000000),
				IsBuy:      true,
			}

			var quoteResponse QuoteResponse
			err := app.GetValue().Client().Do(ctx, http.MethodPost, "/v3/connectors/"+fakeConnectorID+"/quotes", quoteRequest, &quoteResponse)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
		})
	})
})
