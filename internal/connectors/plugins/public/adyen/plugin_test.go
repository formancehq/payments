package adyen

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Adyen Plugin Suite")
}

var _ = Describe("Adyen Plugin", func() {
	var (
		plg    *Plugin
		m      *client.MockClient
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &Plugin{}
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
	})

	Context("install", func() {
		It("reports validation errors in the config - apiKey", func(ctx SpecContext) {
			_, err := New("adyen", logger, json.RawMessage(`{"companyID":"test"}`))
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
		It("reports validation errors in the config - companyID", func(ctx SpecContext) {
			_, err := New("adyen", logger, json.RawMessage(`{"apiKey":"test"}`))
			Expect(err.Error()).To(ContainSubstring("CompanyID"))
		})
		It("returns validation errors when username contains forbidden characters", func(ctx SpecContext) {
			_, err := New("adyen", logger, json.RawMessage(`{"apiKey":"test","companyID": "test","webhookUsername":"some:val"}`))
			Expect(err.Error()).To(ContainSubstring("WebhookUsername"))
		})
		It("returns valid install response without optional parameters", func(ctx SpecContext) {
			_, err := New("adyen", logger, json.RawMessage(`{"apiKey":"test","companyID": "test"}`))
			Expect(err).To(BeNil())
			req := models.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
		It("returns valid install response with all parameters", func(ctx SpecContext) {
			_, err := New("adyen", logger, json.RawMessage(`{"apiKey":"test","companyID": "test", "webhookUsername":"user","webhookPassword":"testvalue","liveEndpointPrefix":"1797a841fbb37ca7"}`))
			Expect(err).To(BeNil())
			req := models.InstallRequest{}
			res, err := plg.Install(ctx, req)
			Expect(err).To(BeNil())
			Expect(len(res.Workflow) > 0).To(BeTrue())
			Expect(res.Workflow).To(Equal(workflow()))
		})
	})

	Context("uninstall", func() {
		It("should return a valid uninstall response when client is set", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "test"}

			m.EXPECT().DeleteWebhook(gomock.Any(), req.ConnectorID).Return(nil)

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

	Context("create transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.CreateTransferRequest{}
			_, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("reverse transfer", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReverseTransferRequest{}
			_, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("poll transfer status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollTransferStatusRequest{}
			_, err := plg.PollTransferStatus(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("create payout", func() {
		It("should fail if client is not set", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{}
			_, err := plg.CreatePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("reverse payout", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.ReversePayoutRequest{}
			_, err := plg.ReversePayout(ctx, req)
			Expect(err).To(MatchError(plugins.ErrNotImplemented.Error()))
		})
	})

	Context("poll payout status", func() {
		It("should fail because not implemented", func(ctx SpecContext) {
			req := models.PollPayoutStatusRequest{}
			_, err := plg.PollPayoutStatus(ctx, req)
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
