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
	state, err := p.fetchAccountEnrichmentData(t.Context())
	if err != nil {
		t.Fatalf("fetchAccountEnrichmentData: %v", err)
	}

	if len(state.markets) != 1 || len(state.myMarkets) != 1 ||
		len(state.tradingFees) != 1 || len(state.withdrawalFees) != 1 {
		t.Errorf("not fully populated: markets=%d myMarkets=%d tradingFees=%d withdrawalFees=%d",
			len(state.markets), len(state.myMarkets),
			len(state.tradingFees), len(state.withdrawalFees))
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
	state, err := p.fetchAccountEnrichmentData(t.Context())
	if err == nil {
		t.Fatal("expected ErrPartialEnrichment, got nil")
	}
	if !errors.Is(err, ErrPartialEnrichment) {
		t.Errorf("expected ErrPartialEnrichment, got %v", err)
	}
	// Successful sources must still be present in the returned state.
	if len(state.markets) != 1 {
		t.Errorf("markets must populate despite my_markets failure, got %+v", state.markets)
	}
}

func TestEnsureEnrichment_DerivativesErrorIsSwallowed(t *testing.T) {
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
	if _, err := p.fetchAccountEnrichmentData(t.Context()); err != nil {
		t.Fatalf("derivatives error must be swallowed, got %v", err)
	}
}

func TestSplitURLSymbol(t *testing.T) {
	t.Parallel()
	currencies := map[string]client.Currency{"BTC": {}, "USD": {}, "USDC": {}, "ETH": {}}
	cases := []struct {
		in    string
		base  string
		quote string
		ok    bool
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
	currencyIndex := map[string]client.Currency{
		"BTC": {Currency: "BTC", Networks: []client.CurrencyNetwork{{Network: "bitcoin"}, {Network: "xrpl"}}},
		"USD": {Currency: "USD"},
		"ETH": {Currency: "ETH"},
		"EUR": {Currency: "EUR"},
		"USDC": {Currency: "USDC"},
	}
	enrich := enrichmentState{
		markets: []client.Market{
			{BaseCurrency: "BTC", CounterCurrency: "USD", MarketType: "SPOT", MinimumOrderValue: "10"},
		},
		myMarkets: []client.MyMarket{
			{Name: "BTC/USD", URLSymbol: "btcusd"},
			{Name: "ETH/USD", URLSymbol: "ethusd"},
		},
		tradingFees: []client.TradingFee{
			{CurrencyPair: "btcusd", Fees: client.TradingFeeRate{Maker: "0.300", Taker: "0.400"}},
		},
		withdrawalFees: []client.WithdrawalFee{
			{Currency: "btc", Network: "bitcoin", Fee: "0.00008"},
			{Currency: "eth", Network: "ethereum", Fee: "0.001"},
		},
	}

	got := buildEnrichmentForCurrency(enrich, currencyIndex, "BTC")

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

// TestBuildEnrichmentForCurrency_DeterministicRepresentative locks the
// representative-pick contract: regardless of source order, the same
// (BaseCurrency-matching) row with the lexicographically smallest
// (CounterCurrency|MarketType) wins. This prevents account-metadata
// flapping across cycles if the PSP returns the slice reordered.
func TestBuildEnrichmentForCurrency_DeterministicRepresentative(t *testing.T) {
	t.Parallel()
	currencyIndex := map[string]client.Currency{
		"BTC": {Currency: "BTC"}, "USD": {Currency: "USD"},
		"EUR": {Currency: "EUR"}, "USDT": {Currency: "USDT"},
	}

	markets := []client.Market{
		{BaseCurrency: "BTC", CounterCurrency: "USDT", MarketType: "SPOT", MinimumOrderValue: "20"},
		{BaseCurrency: "BTC", CounterCurrency: "EUR", MarketType: "SPOT", MinimumOrderValue: "15"},
		{BaseCurrency: "BTC", CounterCurrency: "USD", MarketType: "SPOT", MinimumOrderValue: "10"},
	}
	fees := []client.TradingFee{
		{CurrencyPair: "btcusdt", Fees: client.TradingFeeRate{Maker: "0.500", Taker: "0.600"}},
		{CurrencyPair: "btceur", Fees: client.TradingFeeRate{Maker: "0.300", Taker: "0.400"}},
		{CurrencyPair: "btcusd", Fees: client.TradingFeeRate{Maker: "0.100", Taker: "0.200"}},
	}

	want := struct{ minOrder, makerFee, takerFee string }{
		minOrder: "15", makerFee: "0.300", takerFee: "0.400", // BTC/EUR wins by stable key
	}

	for _, perm := range [][]int{{0, 1, 2}, {2, 1, 0}, {1, 0, 2}, {0, 2, 1}} {
		enrich := enrichmentState{
			markets:     []client.Market{markets[perm[0]], markets[perm[1]], markets[perm[2]]},
			tradingFees: []client.TradingFee{fees[perm[0]], fees[perm[1]], fees[perm[2]]},
		}
		got := buildEnrichmentForCurrency(enrich, currencyIndex, "BTC")
		if got.MinOrderValue != want.minOrder {
			t.Errorf("perm %v: MinOrderValue=%q want %q", perm, got.MinOrderValue, want.minOrder)
		}
		if got.MakerFee != want.makerFee || got.TakerFee != want.takerFee {
			t.Errorf("perm %v: fees maker=%q taker=%q want %q/%q",
				perm, got.MakerFee, got.TakerFee, want.makerFee, want.takerFee)
		}
	}
}
