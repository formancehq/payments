package bitstamp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bitstamp Plugin Suite")
}

var _ = Describe("Bitstamp Plugin", func() {
	var (
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: plugins.NewBasePlugin(),
		}
	})

	Context("install", func() {
		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiSecret": "test-secret"}`)
			_, err := New("bitstamp", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - apiSecret", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test-key"}`)
			_, err := New("bitstamp", logger, config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			p := &Plugin{
				Plugin: plugins.NewBasePlugin(),
				logger: logger,
			}
			req := models.InstallRequest{}
			res, err := p.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("currencies cache", func() {
		It("should not refresh fresh currencies", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin:       plugins.NewBasePlugin(),
				client:       m,
				logger:       logger,
				currencies:   map[string]int{"BTC": 8},
				currLastSync: time.Now(),
			}

			currencies, err := p.getCurrencies(ctx)
			Expect(err).To(BeNil())
			Expect(currencies).To(Equal(map[string]int{"BTC": 8}))
		})

		It("should refresh stale currencies", func(ctx SpecContext) {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			m := client.NewMockClient(ctrl)
			p := &Plugin{
				Plugin:       plugins.NewBasePlugin(),
				client:       m,
				logger:       logger,
				currencies:   map[string]int{"BTC": 8},
				currLastSync: time.Now().Add(-currencyRefreshInterval - time.Minute),
			}

			m.EXPECT().GetCurrencies(gomock.Any()).Return(
				[]client.Currency{
					{Name: "Ethereum", Currency: "ETH", Decimals: 18, Type: "crypto"},
				},
				nil,
			)

			currencies, err := p.getCurrencies(ctx)
			Expect(err).To(BeNil())
			Expect(currencies).To(Equal(map[string]int{"ETH": 18}))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}
			resp, err := plg.Uninstall(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.UninstallResponse{}))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next orders", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextOrdersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOrders(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next conversions", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextConversionsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextConversions(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("capabilities", func() {
		It("declares fetch accounts, balances, payments, orders, conversions", func() {
			Expect(capabilities).To(ContainElements(
				models.CAPABILITY_FETCH_ACCOUNTS,
				models.CAPABILITY_FETCH_BALANCES,
				models.CAPABILITY_FETCH_PAYMENTS,
				models.CAPABILITY_FETCH_ORDERS,
				models.CAPABILITY_FETCH_CONVERSIONS,
			))
		})
	})

	Context("workflow", func() {
		It("has four periodic roots with fetch_orders nested under fetch_accounts", func() {
			tree := workflow()
			Expect(tree).To(HaveLen(4), "expected 4 root tasks (accounts/balances/payments/conversions)")

			rootTypes := make([]models.TaskType, len(tree))
			for i, n := range tree {
				rootTypes[i] = n.TaskType
				Expect(n.Periodically).To(BeTrue(), "task %s should be periodic", n.Name)
			}
			Expect(rootTypes).To(ConsistOf(
				models.TASK_FETCH_ACCOUNTS,
				models.TASK_FETCH_BALANCES,
				models.TASK_FETCH_PAYMENTS,
				models.TASK_FETCH_CONVERSIONS,
			))

			// fetch_orders is a periodic child of fetch_accounts so it
			// receives the parent PSPAccount (and its tradeable markets
			// metadata) via FromPayload.
			var accountsNode models.ConnectorTaskTree
			for _, n := range tree {
				if n.TaskType == models.TASK_FETCH_ACCOUNTS {
					accountsNode = n
					break
				}
			}
			Expect(accountsNode.NextTasks).To(HaveLen(1))
			child := accountsNode.NextTasks[0]
			Expect(child.TaskType).To(Equal(models.TASK_FETCH_ORDERS))
			Expect(child.Periodically).To(BeTrue())
		})
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("fetch next external accounts", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create bank account", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("reverse transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("poll transfer status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("reverse payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("poll payout status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create webhooks", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("translate webhook", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})
})
