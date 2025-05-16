package registry

import (
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Wrapper", func() {
	var (
		ctrl        *gomock.Controller
		plg         *models.MockPlugin
		connectorID models.ConnectorID
		logger      logging.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		plg = models.NewMockPlugin(ctrl)
		connectorID = models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "psp",
		}
		logger = logging.Testing()
	})

	Context("install", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.InstallRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().Install(gomock.Any(), req).Return(models.InstallResponse{}, nil)
			_, err := wrapper.Install(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("uninstall", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.UninstallRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().Uninstall(gomock.Any(), req).Return(models.UninstallResponse{}, nil)
			_, err := wrapper.Uninstall(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("fetch next accounts", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextAccountsRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextAccounts(gomock.Any(), req).Return(models.FetchNextAccountsResponse{}, nil)
			_, err := wrapper.FetchNextAccounts(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("fetch next external accounts", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextExternalAccountsRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextExternalAccounts(gomock.Any(), req).Return(models.FetchNextExternalAccountsResponse{}, nil)
			_, err := wrapper.FetchNextExternalAccounts(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("fetch next payments", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextPaymentsRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextPayments(gomock.Any(), req).Return(models.FetchNextPaymentsResponse{}, nil)
			_, err := wrapper.FetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("fetch next balances", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextBalancesRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextBalances(gomock.Any(), req).Return(models.FetchNextBalancesResponse{}, nil)
			_, err := wrapper.FetchNextBalances(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("fetch next others", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextOthersRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextOthers(gomock.Any(), req).Return(models.FetchNextOthersResponse{}, nil)
			_, err := wrapper.FetchNextOthers(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("create bank account", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CreateBankAccountRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().CreateBankAccount(gomock.Any(), req).Return(models.CreateBankAccountResponse{}, nil)
			_, err := wrapper.CreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("create transfer", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CreateTransferRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().CreateTransfer(gomock.Any(), req).Return(models.CreateTransferResponse{}, nil)
			_, err := wrapper.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("reverse transfer", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.ReverseTransferRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().ReverseTransfer(gomock.Any(), req).Return(models.ReverseTransferResponse{}, nil)
			_, err := wrapper.ReverseTransfer(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("poll transfer status", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.PollTransferStatusRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().PollTransferStatus(gomock.Any(), req).Return(models.PollTransferStatusResponse{}, nil)
			_, err := wrapper.PollTransferStatus(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("create payout", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CreatePayoutRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().CreatePayout(gomock.Any(), req).Return(models.CreatePayoutResponse{}, nil)
			_, err := wrapper.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("reverse payout", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.ReversePayoutRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().ReversePayout(gomock.Any(), req).Return(models.ReversePayoutResponse{}, nil)
			_, err := wrapper.ReversePayout(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("create webhook", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CreateWebhooksRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().CreateWebhooks(gomock.Any(), req).Return(models.CreateWebhooksResponse{}, nil)
			_, err := wrapper.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("verify webhook", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.VerifyWebhookRequest{Config: &models.WebhookConfig{}}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().VerifyWebhook(gomock.Any(), req).Return(models.VerifyWebhookResponse{}, nil)
			_, err := wrapper.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("translate webhook", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.TranslateWebhookRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().TranslateWebhook(gomock.Any(), req).Return(models.TranslateWebhookResponse{}, nil)
			_, err := wrapper.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
		})
	})
})
