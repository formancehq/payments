package bankingcircle

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
	RunSpecs(t, "BankingCircle Plugin Suite")
}

var _ = Describe("BankingCircle Plugin", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("install", func() {
		It("should report errors in config - username", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing username in config: invalid config"))
		})

		It("should report errors in config - password", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"username": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing password in config: invalid config"))
		})

		It("should report errors in config - endpoint", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"username": "test", "password": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing endpoint in config: invalid config"))
		})

		It("should report errors in config - authorization endpoint", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"username": "test", "password": "test", "endpoint": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing authorization endpoint in config: invalid config"))
		})

		It("should report errors in config - certificate", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"username": "test", "password": "test", "endpoint": "test", "authorizationEndpoint": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing user certificate in config: invalid config"))
		})

		It("should report errors in config - certificate key", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"username": "test", "password": "test", "endpoint": "test", "authorizationEndpoint": "test", "userCertificate": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing user certificate key in config: invalid config"))
		})

		It("should report errors in config - invalid certificate", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"username": "test", "password": "test", "endpoint": "test", "authorizationEndpoint": "test", "userCertificate": "test", "userCertificateKey": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("failed to load user certificate: tls: failed to find any PEM data in certificate input: invalid config"))
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
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
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
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented))
		})
	})

	Context("create bank account", func() {
		It("should fail when called before install", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		// Other tests will be in bank_account_creation_test.go
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
