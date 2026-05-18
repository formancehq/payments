package fireblocks

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Fireblocks Plugin Balances", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  *Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{
			logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			client: m,
			assets: map[string]assetInfo{
				"USD": {Asset: "USD/2", Precision: 2, LegacyID: "USD"},
			},
			assetsLastSync: time.Now(),
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("returns error when from payload is missing", func(ctx SpecContext) {
		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("from payload is required"))
		Expect(resp).To(Equal(models.FetchNextBalancesResponse{}))
	})

	It("fetches balances and skips invalid entries", func(ctx SpecContext) {
		from, err := json.Marshal(models.PSPAccount{Reference: "acc-1"})
		Expect(err).To(BeNil())

		m.EXPECT().GetVaultAccount(gomock.Any(), "acc-1").Return(&client.VaultAccount{
			ID: "acc-1",
			Assets: []client.VaultAsset{
				{ID: "USD", Available: "10.50"},
				{ID: "UNKNOWN", Available: "5"},
				{ID: "USD", Available: "bad"},
			},
		}, nil)

		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
			FromPayload: from,
		})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeFalse())
		Expect(resp.Balances).To(HaveLen(1))
		Expect(resp.Balances[0].AccountReference).To(Equal("acc-1"))
		Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(1050)))
		Expect(resp.Balances[0].Asset).To(Equal("USD/2"))
		Expect(resp.Balances[0].CreatedAt.IsZero()).To(BeFalse())
	})

	It("aggregates same-canonical-asset entries across chains", func(ctx SpecContext) {
		// Two distinct Fireblocks legacyIds (USDT_ERC20 / USDT_TRX) both
		// canonicalise to USDT/6 — the plugin must sum them within a vault.
		plg.assets = map[string]assetInfo{
			"USDT_ERC20": {Asset: "USDT/6", Precision: 6, LegacyID: "USDT_ERC20", BlockchainID: "chain-eth"},
			"USDT_TRX":   {Asset: "USDT/6", Precision: 6, LegacyID: "USDT_TRX", BlockchainID: "chain-trx"},
		}

		from, err := json.Marshal(models.PSPAccount{Reference: "acc-1"})
		Expect(err).To(BeNil())

		m.EXPECT().GetVaultAccount(gomock.Any(), "acc-1").Return(&client.VaultAccount{
			ID: "acc-1",
			Assets: []client.VaultAsset{
				{ID: "USDT_ERC20", Available: "100"},
				{ID: "USDT_TRX", Available: "50.5"},
			},
		}, nil)

		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
			FromPayload: from,
		})
		Expect(err).To(BeNil())
		Expect(resp.Balances).To(HaveLen(1))
		Expect(resp.Balances[0].Asset).To(Equal("USDT/6"))
		// 100_000_000 + 50_500_000 = 150_500_000 (6 decimals)
		Expect(resp.Balances[0].Amount).To(Equal(big.NewInt(150500000)))
	})

	It("matches legacyIds case-insensitively (xDAI-style)", func(ctx SpecContext) {
		plg.assets = map[string]assetInfo{
			"XDAI": {Asset: "XDAI/18", Precision: 18, LegacyID: "xDAI"},
		}

		from, err := json.Marshal(models.PSPAccount{Reference: "acc-1"})
		Expect(err).To(BeNil())

		m.EXPECT().GetVaultAccount(gomock.Any(), "acc-1").Return(&client.VaultAccount{
			ID:     "acc-1",
			Assets: []client.VaultAsset{{ID: "xDAI", Available: "1"}},
		}, nil)

		resp, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{
			FromPayload: from,
		})
		Expect(err).To(BeNil())
		Expect(resp.Balances).To(HaveLen(1))
		Expect(resp.Balances[0].Asset).To(Equal("XDAI/18"))
	})
})
