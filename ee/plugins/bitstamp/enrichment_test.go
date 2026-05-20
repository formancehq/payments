package bitstamp

import (
	"errors"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"go.uber.org/mock/gomock"
)

func newTestPlugin(t *testing.T, c client.Client) *Plugin {
	t.Helper()
	return &Plugin{
		name:   "bitstamp",
		client: c,
		logger: logging.NewDefaultLogger(testLogWriter{t}, true, false, false),
		currencies: map[string]int{
			"USD":  2,
			"EUR":  2,
			"BTC":  8,
			"ETH":  18,
			"USDC": 6,
		},
		currLastSync: time.Now(),
	}
}

// testLogWriter sinks logger output to t.Log for visibility on failure.
type testLogWriter struct{ t *testing.T }

func (w testLogWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func TestEnsureEnrichment_HappyPath_PopulatesEveryCache(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := client.NewMockClient(ctrl)
	c.EXPECT().GetMarkets(gomock.Any()).Return([]client.Market{
		{BaseCurrency: "BTC", CounterCurrency: "USD", MarketSymbol: "btcusd", MarketType: "SPOT", MinimumOrderValue: "10"},
	}, nil)
	c.EXPECT().GetMyMarkets(gomock.Any()).Return([]client.MyMarket{
		{Name: "BTC/USD", URLSymbol: "btcusd"},
	}, nil)
	c.EXPECT().GetTradingFees(gomock.Any()).Return([]client.TradingFee{
		{CurrencyPair: "btcusd", Fees: client.TradingFeeRate{Maker: "0.300", Taker: "0.400"}},
	}, nil)
	c.EXPECT().GetWithdrawalFees(gomock.Any()).Return([]client.WithdrawalFee{
		{Currency: "btc", Network: "bitcoin", Fee: "0.00008"},
	}, nil)

	p := newTestPlugin(t, c)
	if err := p.ensureEnrichment(t.Context()); err != nil {
		t.Fatalf("ensureEnrichment: %v", err)
	}

	if len(p.enrichment.markets) != 1 || len(p.enrichment.myMarkets) != 1 ||
		len(p.enrichment.tradingFees) != 1 || len(p.enrichment.withdrawalFees) != 1 {
		t.Errorf("caches not fully populated: markets=%d myMarkets=%d tradingFees=%d withdrawalFees=%d",
			len(p.enrichment.markets), len(p.enrichment.myMarkets),
			len(p.enrichment.tradingFees), len(p.enrichment.withdrawalFees))
	}
}

func TestEnsureEnrichment_PartialFailureReturnsSentinel(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := client.NewMockClient(ctrl)
	c.EXPECT().GetMarkets(gomock.Any()).Return([]client.Market{
		{BaseCurrency: "BTC", CounterCurrency: "USD", MarketSymbol: "btcusd"},
	}, nil)
	c.EXPECT().GetMyMarkets(gomock.Any()).Return(nil, errors.New("503"))
	c.EXPECT().GetTradingFees(gomock.Any()).Return(nil, nil)
	c.EXPECT().GetWithdrawalFees(gomock.Any()).Return(nil, nil)

	p := newTestPlugin(t, c)
	err := p.ensureEnrichment(t.Context())
	if err == nil {
		t.Fatal("expected ErrPartialEnrichment, got nil")
	}
	if !errors.Is(err, ErrPartialEnrichment) {
		t.Errorf("expected ErrPartialEnrichment, got %v", err)
	}
	// Successful caches must still be populated.
	if len(p.enrichment.markets) != 1 {
		t.Errorf("markets cache must populate despite my_markets failure, got %+v", p.enrichment.markets)
	}
}

func TestEnsureEnrichment_TTLShortCircuitsOnFreshCache(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := client.NewMockClient(ctrl)
	// First refresh — explicit call per source.
	c.EXPECT().GetMarkets(gomock.Any()).Return([]client.Market{{MarketSymbol: "btcusd"}}, nil)
	c.EXPECT().GetMyMarkets(gomock.Any()).Return([]client.MyMarket{{URLSymbol: "btcusd"}}, nil)
	c.EXPECT().GetTradingFees(gomock.Any()).Return([]client.TradingFee{{CurrencyPair: "btcusd"}}, nil)
	c.EXPECT().GetWithdrawalFees(gomock.Any()).Return([]client.WithdrawalFee{{Currency: "btc"}}, nil)

	p := newTestPlugin(t, c)
	if err := p.ensureEnrichment(t.Context()); err != nil {
		t.Fatalf("first ensureEnrichment: %v", err)
	}

	// Second call within the TTL window must NOT invoke any client
	// method — the mock controller will fail with "unexpected call"
	// if it does (we set exactly one expectation per method above).
	if err := p.ensureEnrichment(t.Context()); err != nil {
		t.Fatalf("second ensureEnrichment: %v", err)
	}
}

func TestEnsureEnrichment_DerivativesErrorTriggersSkipCache(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c := client.NewMockClient(ctrl)
	c.EXPECT().GetMarkets(gomock.Any()).Return(nil, nil)
	c.EXPECT().GetMyMarkets(gomock.Any()).
		Return(nil, &client.DerivativesUnsupportedError{Endpoint: "/api/v2/my_markets/", Message: "no permission"})
	c.EXPECT().GetTradingFees(gomock.Any()).Return(nil, nil)
	c.EXPECT().GetWithdrawalFees(gomock.Any()).Return(nil, nil)

	p := newTestPlugin(t, c)
	// Derivatives error must be swallowed (returned nil) so the
	// install / cycle is not blocked.
	if err := p.ensureEnrichment(t.Context()); err != nil {
		t.Fatalf("derivatives error must be swallowed, got %v", err)
	}
	if !p.shouldSkipEndpoint("/api/v2/my_markets/") {
		t.Error("my_markets must be flagged in the skip cache")
	}
}

func TestSplitURLSymbol(t *testing.T) {
	t.Parallel()
	currencies := map[string]int{"BTC": 8, "USD": 2, "USDC": 6, "ETH": 18}
	cases := []struct {
		in   string
		base string
		quote string
		ok   bool
	}{
		{"btcusd", "BTC", "USD", true},
		{"btcusdc", "BTC", "USDC", true},
		{"ethusd", "ETH", "USD", true},
		{"", "", "", false},
		{"unknown", "", "", false},
		{"toolong", "", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			b, q, ok := splitURLSymbol(tc.in, currencies)
			if ok != tc.ok || b != tc.base || q != tc.quote {
				t.Errorf("splitURLSymbol(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.in, b, q, ok, tc.base, tc.quote, tc.ok)
			}
		})
	}
}

func TestBuildEnrichmentForCurrency(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p := newTestPlugin(t, client.NewMockClient(ctrl))
	p.enrichment.markets = []client.Market{
		{BaseCurrency: "BTC", CounterCurrency: "USD", MarketType: "SPOT", MinimumOrderValue: "10"},
	}
	p.enrichment.myMarkets = []client.MyMarket{
		{Name: "BTC/USD", URLSymbol: "btcusd"},
		{Name: "ETH/USD", URLSymbol: "ethusd"},
	}
	p.enrichment.tradingFees = []client.TradingFee{
		{CurrencyPair: "btcusd", Fees: client.TradingFeeRate{Maker: "0.300", Taker: "0.400"}},
	}
	p.enrichment.withdrawalFees = []client.WithdrawalFee{
		{Currency: "btc", Network: "bitcoin", Fee: "0.00008"},
		{Currency: "eth", Network: "ethereum", Fee: "0.001"},
	}

	got := p.buildEnrichmentForCurrency(p.currencies, client.Currency{
		Currency: "BTC",
		Networks: []client.CurrencyNetwork{{Network: "bitcoin"}, {Network: "xrpl"}},
	}, "BTC")

	if len(got.Networks) != 2 {
		t.Errorf("Networks not preserved: %+v", got.Networks)
	}
	if len(got.WithdrawalFees) != 1 || got.WithdrawalFees[0].Currency != "btc" {
		t.Errorf("WithdrawalFees filter wrong: %+v", got.WithdrawalFees)
	}
	if len(got.TradableMarkets) != 1 || got.TradableMarkets[0].URLSymbol != "btcusd" {
		t.Errorf("TradableMarkets filter wrong: %+v", got.TradableMarkets)
	}
	if got.MakerFee != "0.300" || got.TakerFee != "0.400" {
		t.Errorf("Fees: maker=%q taker=%q", got.MakerFee, got.TakerFee)
	}
	if got.MinOrderValue != "10" || got.MarketType != "SPOT" {
		t.Errorf("Market: min=%q type=%q", got.MinOrderValue, got.MarketType)
	}
}
