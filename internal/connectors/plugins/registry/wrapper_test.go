package registry

import (
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/plugins"
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

	Context("simple delegators", func() {
		It("Name forwards to plugin", func() {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().Name().Return("dummy")
			Expect(wrapper.Name()).To(Equal("dummy"))
		})

		It("IsScheduledForDeletion forwards to plugin", func() {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().IsScheduledForDeletion().Return(true)
			Expect(wrapper.IsScheduledForDeletion()).To(BeTrue())
		})

		It("ScheduleForDeletion forwards to plugin", func() {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().ScheduleForDeletion(true)
			wrapper.ScheduleForDeletion(true)
		})

		It("Config forwards to plugin", func() {
			wrapper := New(connectorID, logger, plg)
			cfg := struct{ Name string }{Name: "c"}
			plg.EXPECT().Config().Return(cfg)
			Expect(wrapper.Config()).To(Equal(cfg))
		})
	})

	Context("fetch next orders", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextOrdersRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextOrders(gomock.Any(), req).Return(models.FetchNextOrdersResponse{}, nil)
			_, err := wrapper.FetchNextOrders(ctx, req)
			Expect(err).To(BeNil())
		})

		It("translates plugin errors", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextOrders(gomock.Any(), gomock.Any()).Return(models.FetchNextOrdersResponse{}, plugins.ErrNotImplemented)
			_, err := wrapper.FetchNextOrders(ctx, models.FetchNextOrdersRequest{})
			Expect(errors.Is(err, plugins.ErrNotImplemented)).To(BeTrue())
		})
	})

	Context("fetch next conversions", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.FetchNextConversionsRequest{}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextConversions(gomock.Any(), req).Return(models.FetchNextConversionsResponse{}, nil)
			_, err := wrapper.FetchNextConversions(ctx, req)
			Expect(err).To(BeNil())
		})

		It("translates plugin errors", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().FetchNextConversions(gomock.Any(), gomock.Any()).Return(models.FetchNextConversionsResponse{}, plugins.ErrNotImplemented)
			_, err := wrapper.FetchNextConversions(ctx, models.FetchNextConversionsRequest{})
			Expect(errors.Is(err, plugins.ErrNotImplemented)).To(BeTrue())
		})
	})

	Context("poll payout status", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.PollPayoutStatusRequest{PayoutID: "po"}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().PollPayoutStatus(gomock.Any(), req).Return(models.PollPayoutStatusResponse{}, nil)
			_, err := wrapper.PollPayoutStatus(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("trim webhook", func() {
		It("calls underlying function", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.TrimWebhookRequest{Config: &models.WebhookConfig{Name: "n"}}
			plg.EXPECT().Name().Return("dummy").MaxTimes(2)
			plg.EXPECT().TrimWebhook(gomock.Any(), req).Return(models.TrimWebhookResponse{}, nil)
			_, err := wrapper.TrimWebhook(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("user lifecycle", func() {
		It("CreateUser forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CreateUserRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().CreateUser(gomock.Any(), req).Return(models.CreateUserResponse{}, nil)
			_, err := wrapper.CreateUser(ctx, req)
			Expect(err).To(BeNil())
		})

		It("CreateUserLink forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CreateUserLinkRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().CreateUserLink(gomock.Any(), req).Return(models.CreateUserLinkResponse{}, nil)
			_, err := wrapper.CreateUserLink(ctx, req)
			Expect(err).To(BeNil())
		})

		It("CompleteUserLink forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CompleteUserLinkRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().CompleteUserLink(gomock.Any(), req).Return(models.CompleteUserLinkResponse{}, nil)
			_, err := wrapper.CompleteUserLink(ctx, req)
			Expect(err).To(BeNil())
		})

		It("UpdateUserLink forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.UpdateUserLinkRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().UpdateUserLink(gomock.Any(), req).Return(models.UpdateUserLinkResponse{}, nil)
			_, err := wrapper.UpdateUserLink(ctx, req)
			Expect(err).To(BeNil())
		})

		It("CompleteUpdateUserLink forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.CompleteUpdateUserLinkRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().CompleteUpdateUserLink(gomock.Any(), req).Return(models.CompleteUpdateUserLinkResponse{}, nil)
			_, err := wrapper.CompleteUpdateUserLink(ctx, req)
			Expect(err).To(BeNil())
		})

		It("DeleteUserConnection forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.DeleteUserConnectionRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().DeleteUserConnection(gomock.Any(), req).Return(models.DeleteUserConnectionResponse{}, nil)
			_, err := wrapper.DeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
		})

		It("DeleteUser forwards", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			req := models.DeleteUserRequest{}
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().DeleteUser(gomock.Any(), req).Return(models.DeleteUserResponse{}, nil)
			_, err := wrapper.DeleteUser(ctx, req)
			Expect(err).To(BeNil())
		})
	})

	Context("error translation", func() {
		It("wraps rate-limit errors", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().FetchNextAccounts(gomock.Any(), gomock.Any()).Return(models.FetchNextAccountsResponse{}, httpwrapper.ErrStatusCodeTooManyRequests)
			_, err := wrapper.FetchNextAccounts(ctx, models.FetchNextAccountsRequest{})
			Expect(errors.Is(err, plugins.ErrUpstreamRatelimit)).To(BeTrue())
		})

		It("wraps invalid-request errors", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			plg.EXPECT().FetchNextPayments(gomock.Any(), gomock.Any()).Return(models.FetchNextPaymentsResponse{}, models.ErrInvalidRequest)
			_, err := wrapper.FetchNextPayments(ctx, models.FetchNextPaymentsRequest{})
			Expect(errors.Is(err, plugins.ErrInvalidClientRequest)).To(BeTrue())
		})

		It("leaves unknown errors untouched", func(ctx SpecContext) {
			wrapper := New(connectorID, logger, plg)
			plg.EXPECT().Name().Return("dummy").AnyTimes()
			custom := errors.New("custom")
			plg.EXPECT().FetchNextBalances(gomock.Any(), gomock.Any()).Return(models.FetchNextBalancesResponse{}, custom)
			_, err := wrapper.FetchNextBalances(ctx, models.FetchNextBalancesRequest{})
			Expect(err).To(Equal(custom))
		})
	})
})

var _ = Describe("Wrapper optional upgrades", func() {
	var (
		ctrl        *gomock.Controller
		connectorID models.ConnectorID
		logger      logging.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		logger = logging.Testing()
	})

	Context("UseAccountLookup", func() {
		It("no-ops when plugin does not implement PluginWithAccountLookup", func() {
			plg := models.NewMockPlugin(ctrl)
			wrapper := New(connectorID, logger, plg)
			// Should not panic and should not call anything on the plugin.
			wrapper.UseAccountLookup(models.NewMockAccountLookup(ctrl))
		})

		It("forwards when plugin implements PluginWithAccountLookup", func() {
			plg := &pluginWithLookup{
				MockPlugin: models.NewMockPlugin(ctrl),
				inner:      models.NewMockPluginWithAccountLookup(ctrl),
			}
			lookup := models.NewMockAccountLookup(ctrl)
			plg.inner.EXPECT().UseAccountLookup(lookup)
			wrapper := New(connectorID, logger, plg)
			wrapper.UseAccountLookup(lookup)
		})
	})

	Context("BootstrapOnInstall", func() {
		It("returns nil when plugin does not implement PluginWithBootstrapOnInstall", func() {
			plg := models.NewMockPlugin(ctrl)
			wrapper := New(connectorID, logger, plg)
			Expect(wrapper.BootstrapOnInstall()).To(BeNil())
		})

		It("forwards when plugin implements PluginWithBootstrapOnInstall", func() {
			plg := &pluginWithBootstrap{
				MockPlugin: models.NewMockPlugin(ctrl),
				inner:      models.NewMockPluginWithBootstrapOnInstall(ctrl),
			}
			want := []models.TaskType{models.TASK_FETCH_ACCOUNTS}
			plg.inner.EXPECT().BootstrapOnInstall().Return(want)
			wrapper := New(connectorID, logger, plg)
			Expect(wrapper.BootstrapOnInstall()).To(Equal(want))
		})
	})
})

// pluginWithLookup composes MockPlugin with the optional AccountLookup upgrade
// so the type assertion inside wrapper.UseAccountLookup succeeds.
type pluginWithLookup struct {
	*models.MockPlugin
	inner *models.MockPluginWithAccountLookup
}

func (p *pluginWithLookup) UseAccountLookup(lookup models.AccountLookup) {
	p.inner.UseAccountLookup(lookup)
}

// pluginWithBootstrap composes MockPlugin with the optional BootstrapOnInstall
// upgrade so the type assertion inside wrapper.BootstrapOnInstall succeeds.
type pluginWithBootstrap struct {
	*models.MockPlugin
	inner *models.MockPluginWithBootstrapOnInstall
}

func (p *pluginWithBootstrap) BootstrapOnInstall() []models.TaskType {
	return p.inner.BootstrapOnInstall()
}
