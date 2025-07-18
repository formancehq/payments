package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	pluginsError "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plugin Update User Link", func() {
	var (
		act            activities.Activities
		p              *plugins.MockPlugins
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.UpdateUserLinkResponse
		logger         = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay          = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.UpdateUserLinkResponse{
			Link: "https://example.com/update-auth-link",
			TemporaryLinkToken: &models.Token{
				Token:     "temp-update-token-123",
				ExpiresAt: time.Now().Add(time.Hour),
			},
		}
	})

	Context("plugin update user link", func() {
		var (
			plugin *models.MockPlugin
			req    activities.UpdateUserLinkRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = plugins.NewMockPlugins(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			req = activities.UpdateUserLinkRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
				Req: models.UpdateUserLinkRequest{
					AttemptID: "test-update-attempt-id",
					PaymentServiceUser: &models.PSPPaymentServiceUser{
						ID: uuid.New(),
					},
					PSUBankBridge: &models.PSUBankBridge{
						ConnectorID: models.ConnectorID{
							Provider: "some_provider",
						},
					},
					Connection: &models.PSUBankBridgeConnection{
						ConnectionID: "test-connection-id",
					},
					ClientRedirectURL:   stringPtr("https://client.com/update-callback"),
					FormanceRedirectURL: stringPtr("https://formance.com/update-callback"),
					CallBackState:       "test-update-callback-state",
					WebhookBaseURL:      "https://webhook.example.com",
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginUpdateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginUpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error for invalid client request", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrInvalidClientRequest))
			_, err := act.PluginUpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeInvalidArgument))
		})

		It("returns a non-retryable temporal error for not implemented", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, req.Req).Return(sampleResponse, fmt.Errorf("not implemented: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginUpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})

		It("returns error when plugin not found", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(nil, fmt.Errorf("plugin not found"))
			_, err := act.PluginUpdateUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("handles request with minimal required fields", func(ctx SpecContext) {
			minimalReq := activities.UpdateUserLinkRequest{
				ConnectorID: models.ConnectorID{
					Provider: "minimal_provider",
				},
				Req: models.UpdateUserLinkRequest{
					AttemptID: "minimal-update-attempt-id",
					PaymentServiceUser: &models.PSPPaymentServiceUser{
						ID: uuid.New(),
					},
					PSUBankBridge: &models.PSUBankBridge{
						ConnectorID: models.ConnectorID{
							Provider: "minimal_provider",
						},
					},
					Connection: &models.PSUBankBridgeConnection{
						ConnectionID: "minimal-connection-id",
					},
					CallBackState:  "minimal-callback-state",
					WebhookBaseURL: "https://webhook.example.com",
				},
			}
			p.EXPECT().Get(minimalReq.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, minimalReq.Req).Return(sampleResponse, nil)
			res, err := act.PluginUpdateUserLink(ctx, minimalReq)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("handles request with nil optional fields", func(ctx SpecContext) {
			reqWithNilFields := activities.UpdateUserLinkRequest{
				ConnectorID: models.ConnectorID{
					Provider: "nil_fields_provider",
				},
				Req: models.UpdateUserLinkRequest{
					AttemptID: "nil-fields-update-attempt-id",
					PaymentServiceUser: &models.PSPPaymentServiceUser{
						ID: uuid.New(),
					},
					PSUBankBridge: &models.PSUBankBridge{
						ConnectorID: models.ConnectorID{
							Provider: "nil_fields_provider",
						},
					},
					Connection: &models.PSUBankBridgeConnection{
						ConnectionID: "nil-fields-connection-id",
					},
					// ClientRedirectURL and FormanceRedirectURL are nil
					CallBackState:  "nil-fields-callback-state",
					WebhookBaseURL: "https://webhook.example.com",
				},
			}
			p.EXPECT().Get(reqWithNilFields.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, reqWithNilFields.Req).Return(sampleResponse, nil)
			res, err := act.PluginUpdateUserLink(ctx, reqWithNilFields)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("handles response with nil temporary link token", func(ctx SpecContext) {
			responseWithNilToken := models.UpdateUserLinkResponse{
				Link: "https://example.com/update-auth-link-no-token",
				// TemporaryLinkToken is nil
			}
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().UpdateUserLink(ctx, req.Req).Return(responseWithNilToken, nil)
			res, err := act.PluginUpdateUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&responseWithNilToken))
			Expect(res.TemporaryLinkToken).To(BeNil())
		})
	})
})
