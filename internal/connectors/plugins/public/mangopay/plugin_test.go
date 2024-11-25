package mangopay

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mangopay Plugin Suite")
}

var _ = Describe("Mangopay Plugin", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("install", func() {
		It("should report errors in config - clientID", func(ctx SpecContext) {
			config := json.RawMessage(`{"apiKey": "test", "endpoint": "test"}`)
			_, err := New("mangopay", config)
			Expect(err).To(MatchError("missing clientID in config: invalid config"))
		})

		It("should report errors in config - apiKey", func(ctx SpecContext) {
			config := json.RawMessage(`{"clientID": "test", "endpoint": "test"}`)
			_, err := New("mangopay", config)
			Expect(err).To(MatchError("missing api key in config: invalid config"))
		})

		It("should report errors in config - endpoint", func(ctx SpecContext) {
			config := json.RawMessage(`{"clientID": "test", "apiKey": "test"}`)
			_, err := New("mangopay", config)
			Expect(err).To(MatchError("missing endpoint in config: invalid config"))
		})

		It("should return valid install response", func(ctx SpecContext) {
			config := json.RawMessage(`{"clientID": "test", "apiKey": "test", "endpoint": "test"}`)
			_, err := New("mangopay", config)
			Expect(err).To(BeNil())
			req := models.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
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

		// Other tests will be in accounts_test.go
	})

	Context("fetch next balances", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in balances_test.go
	})

	Context("fetch next external accounts", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in external_accounts_test.go
	})

	Context("fetch next payments", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in payments_test.go
	})

	Context("fetch next others", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in others_test.go
	})

	Context("create bank account", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in bank_accounts_test.go
	})

	Context("create transfer", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in transfers_test.go
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
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in payouts_test.go
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
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("translate webhook", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})
})
