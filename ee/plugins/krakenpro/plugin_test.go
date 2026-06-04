package krakenpro

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kraken Pro Plugin Suite")
}

var _ = Describe("Kraken Pro Plugin", func() {
	var (
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
		}
	})

	Context("config validation", func() {
		It("rejects missing apiKey", func() {
			_, err := New("krakenpro", logger, json.RawMessage(`{"apiSecret":"AAAA"}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("rejects missing apiSecret", func() {
			_, err := New("krakenpro", logger, json.RawMessage(`{"apiKey":"x"}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})

		It("rejects non-base64 apiSecret", func() {
			_, err := New("krakenpro", logger, json.RawMessage(`{"apiKey":"x","apiSecret":"@@@not-base64@@@","endpoint":"https://api.uat.kraken.com"}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("decode"))
		})

		It("accepts a valid config with base64 secret", func() {
			_, err := New("krakenpro", logger, json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA==","endpoint":"https://api.uat.kraken.com"}`))
			Expect(err).To(BeNil())
		})
	})

	Context("install", func() {
		It("calls BalanceEx for credential validation and returns workflow", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			m := client.NewMockClient(ctrl)
			m.EXPECT().GetBalanceEx(gomock.Any()).Return(map[string]client.BalanceExEntry{}, nil)
			m.EXPECT().GetAssets(gomock.Any()).Return(map[string]client.AssetInfo{
				"XXBT": {Altname: "XBT", Decimals: 8},
			}, nil)
			m.EXPECT().GetAssetPairs(gomock.Any()).Return(map[string]client.AssetPair{}, nil)

			p := &Plugin{Plugin: plugins.NewBasePlugin(), client: m, logger: logger}
			resp, err := p.Install(ctx, models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(resp.Workflow).To(Equal(workflow()))
		})

		It("fails fast (ErrInvalidRequest) on EAPI:Invalid key so Temporal stops retrying", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			m := client.NewMockClient(ctrl)
			m.EXPECT().GetBalanceEx(gomock.Any()).Return(nil, &client.APIError{Endpoint: "/0/private/BalanceEx", Code: "EAPI:Invalid key", All: []string{"EAPI:Invalid key"}})

			p := &Plugin{Plugin: plugins.NewBasePlugin(), client: m, logger: logger}
			_, err := p.Install(ctx, models.InstallRequest{})
			Expect(err).To(MatchError(models.ErrInvalidRequest))
		})

		It("fails fast (ErrInvalidRequest) on EAPI:Invalid nonce — preserves the nonce window", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			m := client.NewMockClient(ctrl)
			m.EXPECT().GetBalanceEx(gomock.Any()).Return(nil, &client.APIError{Endpoint: "/0/private/BalanceEx", Code: "EAPI:Invalid nonce", All: []string{"EAPI:Invalid nonce"}})

			p := &Plugin{Plugin: plugins.NewBasePlugin(), client: m, logger: logger}
			_, err := p.Install(ctx, models.InstallRequest{})
			Expect(err).To(MatchError(models.ErrInvalidRequest))
		})

		It("bubbles up retryable errors (EService:Unavailable) for Temporal to retry", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			m := client.NewMockClient(ctrl)
			m.EXPECT().GetBalanceEx(gomock.Any()).Return(nil, &client.APIError{Endpoint: "/0/private/BalanceEx", Code: "EService:Unavailable", All: []string{"EService:Unavailable"}})

			p := &Plugin{Plugin: plugins.NewBasePlugin(), client: m, logger: logger}
			_, err := p.Install(ctx, models.InstallRequest{})
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(models.ErrInvalidRequest))
		})
	})

	Context("BootstrapOnInstall", func() {
		It("declares TASK_FETCH_ACCOUNTS as the bootstrap step", func() {
			Expect(plg.BootstrapOnInstall()).To(Equal([]models.TaskType{models.TASK_FETCH_ACCOUNTS}))
		})
	})

	Context("UseAccountLookup", func() {
		It("wires the lookup once", func() {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			lookup := models.NewMockAccountLookup(ctrl)
			plg.UseAccountLookup(lookup)
			Expect(plg.accountLookup).To(Equal(models.AccountLookup(lookup)))
		})
	})

	Context("FetchNext* pre-install guards", func() {
		It("FetchNextAccounts returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{State: json.RawMessage(`{}`)})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("FetchNextBalances returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("FetchNextPayments returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("FetchNextOrders returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.FetchNextOrders(ctx, models.FetchNextOrdersRequest{State: json.RawMessage(`{}`)})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
		It("FetchNextConversions returns ErrNotYetInstalled", func(ctx SpecContext) {
			_, err := plg.FetchNextConversions(ctx, models.FetchNextConversionsRequest{State: json.RawMessage(`{}`)})
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("Uninstall", func() {
		It("is a no-op", func(ctx SpecContext) {
			resp, err := plg.Uninstall(ctx, models.UninstallRequest{ConnectorID: "test"})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})

	Context("capabilities", func() {
		It("declares all five fetch-only capabilities", func() {
			Expect(capabilities).To(ConsistOf(
				models.CAPABILITY_FETCH_ACCOUNTS,
				models.CAPABILITY_FETCH_BALANCES,
				models.CAPABILITY_FETCH_PAYMENTS,
				models.CAPABILITY_FETCH_ORDERS,
				models.CAPABILITY_FETCH_CONVERSIONS,
			))
		})
	})

	Context("workflow", func() {
		It("nests fetch_orders under fetch_accounts and exposes 4 roots", func() {
			tree := workflow()
			Expect(tree).To(HaveLen(4))
			rootTypes := make([]models.TaskType, len(tree))
			var accountsNode models.ConnectorTaskTree
			for i, n := range tree {
				rootTypes[i] = n.TaskType
				Expect(n.Periodically).To(BeTrue(), "root %s must be periodic", n.Name)
				if n.TaskType == models.TASK_FETCH_ACCOUNTS {
					accountsNode = n
				}
			}
			Expect(rootTypes).To(ConsistOf(
				models.TASK_FETCH_ACCOUNTS,
				models.TASK_FETCH_BALANCES,
				models.TASK_FETCH_PAYMENTS,
				models.TASK_FETCH_CONVERSIONS,
			))
			Expect(accountsNode.NextTasks).To(HaveLen(1))
			Expect(accountsNode.NextTasks[0].TaskType).To(Equal(models.TASK_FETCH_ORDERS))
			Expect(accountsNode.NextTasks[0].Periodically).To(BeTrue())
		})
	})

	Context("asset cache TTL", func() {
		It("does not refresh a fresh cache", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin:       plugins.NewBasePlugin(),
				client:       m,
				logger:       logger,
				currencies:   map[string]int{"BTC": 8},
				assetsLoaded: time.Now(),
			}
			currencies, _, err := p.ensureAssets(ctx)
			Expect(err).To(BeNil())
			Expect(currencies).To(Equal(map[string]int{"BTC": 8}))
		})

		It("refreshes a stale cache", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			m := client.NewMockClient(ctrl)
			m.EXPECT().GetAssets(gomock.Any()).Return(map[string]client.AssetInfo{
				"XETH": {Altname: "ETH", Decimals: 18},
			}, nil)
			m.EXPECT().GetAssetPairs(gomock.Any()).Return(map[string]client.AssetPair{}, nil)
			p := &Plugin{
				Plugin:       plugins.NewBasePlugin(),
				client:       m,
				logger:       logger,
				currencies:   map[string]int{"BTC": 8},
				assetsLoaded: time.Now().Add(-assetRefreshTTL - time.Minute),
			}
			currencies, _, err := p.ensureAssets(ctx)
			Expect(err).To(BeNil())
			Expect(currencies).To(HaveKeyWithValue("ETH", 18))
		})
	})
})
