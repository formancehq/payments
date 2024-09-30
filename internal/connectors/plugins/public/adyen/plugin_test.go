package adyen

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stripe Plugin Suite")
}

var _ = Describe("Stripe Plugin", func() {
	var (
		plg *Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		plg = &Plugin{}
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
	})

	Context("install", func() {
		It("reports validation errors in the config - apiKey", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"companyID": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing apiKey in config: invalid config"))
		})
		It("reports validation errors in the config - companyID", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"apiKey": "test"}`)}
			_, err := plg.Install(ctx, req)
			Expect(err).To(MatchError("missing companyID in config: invalid config"))
		})
		It("returns valid install response", func(ctx SpecContext) {
			req := models.InstallRequest{Config: json.RawMessage(`{"apiKey":"test", "companyID": "test"}`)}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Capabilities) > 0).To(BeTrue())
			Expect(res.Capabilities).To(Equal(capabilities))
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return a valid uninstall response when client is set", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}

			m.EXPECT().DeleteWebhook(ctx, req.ConnectorID).Return(nil)

			plg := &Plugin{client: m}
			_, err := plg.Uninstall(ctx, req)
			Expect(err).To(BeNil())
		})
		It("should fail if client is not set", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}

			_, err := plg.Uninstall(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})

	Context("fetch next accounts", func() {
		It("should fail if client is not set", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})

	Context("fetch next balances", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextBalancesRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextBalances(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("fetch next external accounts", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextExternalAccountsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("fetch next payments", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextPayments(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("fetch next others", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.FetchNextOthersRequest{State: json.RawMessage(`{}`)}
			_, err := plg.FetchNextOthers(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("create bank account", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateBankAccountRequest{}
			_, err := plg.CreateBankAccount(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("create webhooks", func() {
		It("should fail because not yet installed", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})

	Context("translate webhook", func() {
		It("should fail because not yet installed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled.Error()))
		})
	})
})
