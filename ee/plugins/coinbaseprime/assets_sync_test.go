package coinbaseprime

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Coinbase Plugin ensureAssetsFresh", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("is a no-op within the TTL window after a recent sync", func(ctx SpecContext) {
		plg.currencies = map[string]int{"USD": 2}
		plg.networkSymbols = map[string]string{}
		plg.entityID = "entity-fresh"
		plg.assetsLastSync = time.Now()

		// No GetPortfolio / GetAssets expectations -- any call would fail the test.
		Expect(plg.ensureAssetsFresh(ctx)).To(Succeed())
	})

	It("triggers a refresh when the cache is empty", func(ctx SpecContext) {
		m.EXPECT().GetPortfolio(gomock.Any()).Return(
			&client.PortfolioResponse{
				Portfolio: client.Portfolio{EntityID: "entity-empty"},
			},
			nil,
		)
		m.EXPECT().GetAssets(gomock.Any(), "entity-empty").Return(
			&client.AssetsResponse{
				Assets: []client.Asset{{Symbol: "BTC", DecimalPrecision: "8"}},
			},
			nil,
		)

		Expect(plg.ensureAssetsFresh(ctx)).To(Succeed())
		Expect(plg.currencies).To(HaveKey("BTC"))
		Expect(plg.entityID).To(Equal("entity-empty"))
		Expect(plg.assetsLastSync).ToNot(BeZero())
	})

	It("triggers a refresh when the cache is stale beyond the TTL", func(ctx SpecContext) {
		plg.currencies = map[string]int{"USD": 2}
		plg.networkSymbols = map[string]string{}
		plg.entityID = "entity-stale"
		plg.assetsLastSync = time.Now().Add(-2 * assetRefreshInterval)

		// Portfolio is already known -- only GetAssets should be called.
		m.EXPECT().GetAssets(gomock.Any(), "entity-stale").Return(
			&client.AssetsResponse{
				Assets: []client.Asset{{Symbol: "BTC", DecimalPrecision: "8"}},
			},
			nil,
		)

		before := plg.assetsLastSync
		Expect(plg.ensureAssetsFresh(ctx)).To(Succeed())
		Expect(plg.currencies).To(HaveKey("BTC"))
		Expect(plg.assetsLastSync.After(before)).To(BeTrue())
	})

	It("does not re-fetch the portfolio once entityID is cached", func(ctx SpecContext) {
		plg.currencies = map[string]int{}
		plg.entityID = "entity-cached"
		plg.assetsLastSync = time.Time{} // stale

		// Only GetAssets -- no GetPortfolio mock means the test fails if it's called.
		m.EXPECT().GetAssets(gomock.Any(), "entity-cached").Return(
			&client.AssetsResponse{
				Assets: []client.Asset{{Symbol: "USDC", DecimalPrecision: "6"}},
			},
			nil,
		)

		Expect(plg.ensureAssetsFresh(ctx)).To(Succeed())
	})

	It("collapses concurrent callers into a single GetAssets call", func(ctx SpecContext) {
		plg.entityID = "entity-concurrent"
		plg.assetsLastSync = time.Time{} // stale

		// Gate GetAssets so both goroutines race past the fast-path check before
		// the first one completes. Exactly one call is expected.
		gate := make(chan struct{})
		m.EXPECT().GetAssets(gomock.Any(), "entity-concurrent").DoAndReturn(
			func(_ context.Context, _ string) (*client.AssetsResponse, error) {
				<-gate
				return &client.AssetsResponse{
					Assets: []client.Asset{{Symbol: "BTC", DecimalPrecision: "8"}},
				}, nil
			},
		).Times(1)

		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- plg.ensureAssetsFresh(ctx)
			}()
		}
		// Small yield so both goroutines can enter ensureAssetsFresh.
		time.Sleep(10 * time.Millisecond)
		close(gate)
		wg.Wait()
		close(errs)
		for err := range errs {
			Expect(err).To(Succeed())
		}
	})

	It("propagates GetAssets errors and leaves stale state untouched", func(ctx SpecContext) {
		plg.currencies = map[string]int{"USD": 2}
		plg.entityID = "entity-err"
		plg.assetsLastSync = time.Time{}

		m.EXPECT().GetAssets(gomock.Any(), "entity-err").Return(nil, errors.New("boom"))

		err := plg.ensureAssetsFresh(ctx)
		Expect(err).To(MatchError(ContainSubstring("boom")))
		// Stale timestamp is preserved so next call will retry.
		Expect(plg.assetsLastSync).To(BeZero())
	})
})

var _ = Describe("Coinbase Plugin resolveAssetAndPrecision", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
			client: m,
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			// Pre-populated, TTL-fresh cache. No client mocks below — any
			// unexpected client call fails the test, proving the fast-path
			// served these cases.
			currencies: map[string]int{
				"BTC":  8,
				"USDC": 6,
				"USD":  2,
			},
			networkSymbols: map[string]string{
				"BASEUSDC": "USDC",
			},
			assetsLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("resolves a known symbol to asset and precision", func(ctx SpecContext) {
		asset, precision, ok, err := plg.resolveAssetAndPrecision(ctx, "BTC")
		Expect(err).To(BeNil())
		Expect(ok).To(BeTrue())
		Expect(asset).To(Equal("BTC/8"))
		Expect(precision).To(Equal(8))
	})

	It("folds network-scoped symbols to the base symbol", func(ctx SpecContext) {
		asset, precision, ok, err := plg.resolveAssetAndPrecision(ctx, "BASEUSDC")
		Expect(err).To(BeNil())
		Expect(ok).To(BeTrue())
		Expect(asset).To(Equal("USDC/6"))
		Expect(precision).To(Equal(6))
	})

	It("normalizes whitespace and case before lookup", func(ctx SpecContext) {
		asset, precision, ok, err := plg.resolveAssetAndPrecision(ctx, "  btc  ")
		Expect(err).To(BeNil())
		Expect(ok).To(BeTrue())
		Expect(asset).To(Equal("BTC/8"))
		Expect(precision).To(Equal(8))
	})

	It("returns ok=false with no error for an unknown symbol", func(ctx SpecContext) {
		asset, precision, ok, err := plg.resolveAssetAndPrecision(ctx, "DOGE")
		Expect(err).To(BeNil())
		Expect(ok).To(BeFalse())
		Expect(asset).To(BeEmpty())
		Expect(precision).To(Equal(0))
	})

	It("propagates freshness-refresh errors from getAssets", func(ctx SpecContext) {
		// Force a refresh attempt by making the cache stale; mock GetAssets to fail.
		plg.assetsLastSync = time.Time{}
		plg.entityID = "entity-resolve-err"
		m.EXPECT().GetAssets(gomock.Any(), "entity-resolve-err").Return(nil, errors.New("boom"))

		asset, precision, ok, err := plg.resolveAssetAndPrecision(ctx, "BTC")
		Expect(err).To(MatchError(ContainSubstring("boom")))
		Expect(ok).To(BeFalse())
		Expect(asset).To(BeEmpty())
		Expect(precision).To(Equal(0))
	})
})
