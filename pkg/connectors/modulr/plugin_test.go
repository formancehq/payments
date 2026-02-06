package modulr

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Modulr Plugin Suite")
}

var _ = Describe("Modulr Plugin", func() {
	var (
		plg    *Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{
			Plugin: connector.NewBasePlugin(),
		}
	})

	Context("install", func() {
		It("should report errors in config - apiSecret", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "endpoint": "test"}`)
			_, err := New(ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})

		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiSecret": "test", "endpoint": "test"}`)
			_, err := New(ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})

		It("should report errors in config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiSecret": "test", "apiKey": "test"}`)
			_, err := New(ProviderName, logger, config)
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiSecret": "test", "apiKey": "test", "endpoint": "test"}`)
			_, err := New(ProviderName, logger, config)
			Expect(err).To(BeNil())
			req := connector.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return valid uninstall response", func(ctx SpecContext) {
			req := connector.UninstallRequest{ConnectorID: "test"}
			resp, err := plg.Uninstall(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.UninstallResponse{}))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})

		// Other tests will be in accounts_test.go
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})

		// Other tests will be in balances_test.go
	})

	Context("fetch next external accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})

		// Other tests will be in external_accounts_test.go
	})

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})

		// Other tests will be in payments_test.go
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("create bank account", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("create transfer", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})

		// Other tests will be in transfers_test.go
	})

	Context("reverse transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("poll transfer status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("create payout", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := connector.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotYetInstalled))
		})

		// Other tests will be in payouts_test.go
	})

	Context("reverse payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("poll payout status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("create webhooks", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})

	Context("translate webhook", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := connector.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(connector.ErrNotImplemented))
		})
	})
})
