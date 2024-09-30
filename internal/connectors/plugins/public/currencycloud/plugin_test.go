package currencycloud

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
	RunSpecs(t, "Stripe Plugin Suite")
}

var _ = Describe("Currencycloud Plugin", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("install", func() {
		It("reports validation errors in the config - loginID", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"apiKey": "test", "endpoint": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing clientID in config: invalid config"))
		})

		It("reports validation errors in the config - apiKey", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"loginID": "test", "endpoint": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing api key in config: invalid config"))
		})

		It("reports validation errors in the config - endpoint", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"loginID": "test", "apiKey": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing endpoint in config: invalid config"))
		})

		It("returns valid install response", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"loginID": "test", "apiKey": "test", "endpoint": "test"}`)}

			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Capabilities) > 0).To(BeTrue())
			Expect(res.Capabilities).To(Equal(capabilities))
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
			Expect(len(res.WebhooksConfigs) == 0).To(BeTrue())
		})
	})

	Context("uninstall", func() {
		It("returns valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{
				ConnectorID: "test",
			}

			res, err := plg.Uninstall(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(models.UninstallResponse{}))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail because plugin is not installed", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{}

			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})

	Context("fetch next balances", func() {
		It("should fail because plugin is not installed", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{}

			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})

	Context("fetch next external accounts", func() {
		It("should fail because plugin is not installed", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{}

			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})

	Context("fetch next payments", func() {
		It("should fail because plugin is not installed", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{}

			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})

	Context("fetch next others", func() {
		It("should fail because of unimplemented method", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{}

			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("create bank account", func() {
		It("should fail because of unimplemented method", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}

			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("create webhooks", func() {
		It("should fail because of unimplemented method", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}

			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("translate webhook", func() {
		It("should fail because of unimplemented method", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}

			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})
})
