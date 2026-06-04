package krakenpro

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Kraken Pro fetch_accounts", func() {
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
			currencies: map[string]int{
				"BTC": 8, "USD": 2, "EUR": 2, "ADA": 8,
			},
			assetCodes: map[string]string{
				"BTC": "XXBT", "USD": "ZUSD", "EUR": "ZEUR", "ADA": "ADA",
			},
			assetsLoaded: time.Now(),
		}
	})

	AfterEach(func() { ctrl.Finish() })

	walletType := func(a models.PSPAccount) string {
		return a.Metadata["com.krakenpro.spec/wallet_type"]
	}
	byRef := func(accs []models.PSPAccount) map[string]models.PSPAccount {
		out := map[string]models.PSPAccount{}
		for _, a := range accs {
			out[a.Reference] = a
		}
		return out
	}

	It("emits one PSPAccount per asset class, keyed by raw code", func(ctx SpecContext) {
		// XXBT spot + ZUSD spot + ADA.S staked (no ADA spot row) → the
		// ADA spot account is force-emitted from the /Assets code.
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{
			"XXBT":  {Balance: "1.0", HoldTrade: "0"},
			"ZUSD":  {Balance: "100.00", HoldTrade: "10.00"},
			"ADA.S": {Balance: "5.0", HoldTrade: "0"},
		}, nil)

		resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		Expect(err).To(BeNil())
		Expect(resp.HasMore).To(BeFalse())
		accs := byRef(resp.Accounts)
		Expect(accs).To(HaveKey("XXBT"))
		Expect(accs).To(HaveKey("ZUSD"))
		Expect(accs).To(HaveKey("ADA.S"))
		Expect(accs).To(HaveKey("ADA")) // force-emitted spot
		Expect(walletType(accs["XXBT"])).To(Equal("spot"))
		Expect(walletType(accs["ADA.S"])).To(Equal("staked"))
		Expect(walletType(accs["ADA"])).To(Equal("spot"))
		Expect(*accs["XXBT"].DefaultAsset).To(Equal("BTC/8"))
	})

	It("force-emits a spot account when value sits only in an earn variant", func(ctx SpecContext) {
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{
			"XBT.M": {Balance: "0.3", HoldTrade: "0"},
		}, nil)
		resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		Expect(err).To(BeNil())
		accs := byRef(resp.Accounts)
		Expect(accs).To(HaveKey("XBT.M"))
		Expect(accs).To(HaveKey("XXBT")) // spot, force-emitted at zero
		Expect(walletType(accs["XBT.M"])).To(Equal("rewards"))
		Expect(walletType(accs["XXBT"])).To(Equal("spot"))
	})

	It("skips assets already emitted in a previous cycle (keyed by reference)", func(ctx SpecContext) {
		state := accountsState{AccountAssetsImportedAt: map[string]string{"XXBT": "2025-01-01T00:00:00Z"}}
		stateBytes, _ := json.Marshal(state)

		m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{
			"XXBT": {Balance: "1.0"},
			"ZUSD": {Balance: "100.00"},
		}, nil)

		resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: stateBytes})
		Expect(err).To(BeNil())
		// XXBT already imported (and counts as the BTC spot) → only ZUSD emitted.
		Expect(resp.Accounts).To(HaveLen(1))
		Expect(resp.Accounts[0].Reference).To(Equal("ZUSD"))
	})

	It("propagates BalanceEx errors", func(ctx SpecContext) {
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(nil, errors.New("boom"))
		_, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		Expect(err).To(HaveOccurred())
	})

	It("emits nothing when every row is zero", func(ctx SpecContext) {
		m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{
			"XXBT": {Balance: "0", HoldTrade: "0"},
		}, nil)
		resp, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
		Expect(err).To(BeNil())
		Expect(resp.Accounts).To(BeEmpty())
	})
})
