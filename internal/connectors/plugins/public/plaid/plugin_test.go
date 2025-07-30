package plaid_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/public/plaid"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plaid *Plugin Suite")
}

var _ = Describe("Plaid *Plugin", func() {
	var (
		plg    *plaid.Plugin
		config json.RawMessage
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		plg = &plaid.Plugin{
			Plugin: plugins.NewBasePlugin(),
		}
		config = json.RawMessage(`{"clientID":"1234","clientSecret":"abc123","isSandbox":true}`)
	})

	Context("install", func() {
		It("reports validation errors in the config", func(ctx SpecContext) {
			config := json.RawMessage(`{}`)
			connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "plaid"}
			_, err := plaid.New("plaid", logger, connectorID, config)
			Expect(err.Error()).To(ContainSubstring("validation"))
		})

		It("returns valid install response", func(ctx SpecContext) {
			connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "plaid"}
			_, err := plaid.New("plaid", logger, connectorID, config)
			Expect(err).To(BeNil())
			res, err := plg.Install(context.Background(), models.InstallRequest{})
			Expect(err).To(BeNil())
			Expect(res.Workflow).ToNot(BeEmpty())
		})
	})

	Context("uninstall", func() {
		It("returns valid uninstall response", func(ctx SpecContext) {
			req := models.UninstallRequest{ConnectorID: "dummyID"}
			_, err := plg.Uninstall(context.Background(), req)
			Expect(err).To(BeNil())
		})
	})

	Context("calling functions on uninstalled plugins", func() {
		It("fails when fetch next accounts is called before install", func(ctx SpecContext) {
			req := models.FetchNextAccountsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextAccounts(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when fetch next payments is called before install", func(ctx SpecContext) {
			req := models.FetchNextPaymentsRequest{
				State: json.RawMessage(`{}`),
			}
			_, err := plg.FetchNextPayments(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when create user is called before install", func(ctx SpecContext) {
			req := models.CreateUserRequest{}
			_, err := plg.CreateUser(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when create user link is called before install", func(ctx SpecContext) {
			req := models.CreateUserLinkRequest{}
			_, err := plg.CreateUserLink(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when complete user link is called before install", func(ctx SpecContext) {
			req := models.CompleteUserLinkRequest{}
			_, err := plg.CompleteUserLink(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when update user link is called before install", func(ctx SpecContext) {
			req := models.UpdateUserLinkRequest{}
			_, err := plg.UpdateUserLink(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when delete user is called before install", func(ctx SpecContext) {
			req := models.DeleteUserRequest{}
			_, err := plg.DeleteUser(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when delete user connection is called before install", func(ctx SpecContext) {
			req := models.DeleteUserConnectionRequest{}
			_, err := plg.DeleteUserConnection(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when create webhooks is called before install", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}
			_, err := plg.CreateWebhooks(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when verify webhook is called before install", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{}
			_, err := plg.VerifyWebhook(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})

		It("fails when translate webhook is called before install", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}
			_, err := plg.TranslateWebhook(context.Background(), req)
			Expect(err).To(MatchError(plugins.ErrNotYetInstalled))
		})
	})
})
