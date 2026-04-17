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
