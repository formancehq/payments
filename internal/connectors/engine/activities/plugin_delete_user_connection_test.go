package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	pluginsError "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "github.com/golang/mock/gomock"
)

var _ = Describe("Plugin Delete User Connection", func() {
	var (
		act            activities.Activities
		p              *connectors.MockManager
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.DeleteUserConnectionResponse
		logger         = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay          = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.DeleteUserConnectionResponse{}
	})

	Context("plugin delete user connection", func() {
		var (
			plugin *models.MockPlugin
			req    activities.DeleteUserConnectionRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			req = activities.DeleteUserConnectionRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
				Req: models.DeleteUserConnectionRequest{
					PaymentServiceUser: &models.PSPPaymentServiceUser{
						ID:        uuid.New(),
						Name:      "Test User",
						CreatedAt: time.Now(),
						ContactDetails: &models.ContactDetails{
							Email:       pointer.For("test@example.com"),
							PhoneNumber: pointer.For("+1234567890"),
							Locale:      pointer.For("en-US"),
						},
						Address: &models.Address{
							StreetName:   pointer.For("Test Street"),
							StreetNumber: pointer.For("123"),
							City:         pointer.For("Test City"),
							Region:       pointer.For("Test Region"),
							PostalCode:   pointer.For("12345"),
							Country:      pointer.For("US"),
						},
						Metadata: map[string]string{
							"source": "test",
						},
					},
					OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
						ConnectorID: models.ConnectorID{
							Provider: "some_provider",
						},
						Metadata: map[string]string{
							"open_banking_forwarded_user_id": "test-ob-123",
						},
					},
					Connection: &models.PSPOpenBankingConnection{
						ConnectionID: "external-connection-123",
						CreatedAt:    time.Now(),
						Metadata: map[string]string{
							"connection_id": "external-connection-123",
						},
					},
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUserConnection(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginDeleteUserConnection(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUserConnection(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginDeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error for invalid client request", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUserConnection(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrInvalidClientRequest))
			_, err := act.PluginDeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeInvalidArgument))
		})

		It("returns a non-retryable temporal error for not implemented", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUserConnection(ctx, req.Req).Return(sampleResponse, fmt.Errorf("not implemented: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginDeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})

		It("returns error when plugin not found", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(nil, fmt.Errorf("plugin not found"))
			_, err := act.PluginDeleteUserConnection(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("handles request with minimal required fields", func(ctx SpecContext) {
			minimalReq := activities.DeleteUserConnectionRequest{
				ConnectorID: models.ConnectorID{
					Provider: "minimal_provider",
				},
				Req: models.DeleteUserConnectionRequest{
					PaymentServiceUser: &models.PSPPaymentServiceUser{
						ID:        uuid.New(),
						Name:      "Minimal User",
						CreatedAt: time.Now(),
					},
					OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
						ConnectorID: models.ConnectorID{
							Provider: "minimal_provider",
						},
					},
					Connection: &models.PSPOpenBankingConnection{
						ConnectionID: "minimal-connection-123",
						CreatedAt:    time.Now(),
					},
				},
			}
			p.EXPECT().Get(minimalReq.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUserConnection(ctx, minimalReq.Req).Return(sampleResponse, nil)
			res, err := act.PluginDeleteUserConnection(ctx, minimalReq)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("handles request with nil optional fields", func(ctx SpecContext) {
			reqWithNilFields := activities.DeleteUserConnectionRequest{
				ConnectorID: models.ConnectorID{
					Provider: "nil_fields_provider",
				},
				Req: models.DeleteUserConnectionRequest{
					PaymentServiceUser: &models.PSPPaymentServiceUser{
						ID:        uuid.New(),
						Name:      "Nil Fields User",
						CreatedAt: time.Now(),
						// ContactDetails and Address are nil
					},
					OpenBankingForwardedUser: &models.OpenBankingForwardedUser{
						ConnectorID: models.ConnectorID{
							Provider: "nil_fields_provider",
						},
						// Metadata is nil
					},
					Connection: &models.PSPOpenBankingConnection{
						ConnectionID: "nil-fields-connection-123",
						CreatedAt:    time.Now(),
						// Metadata is nil
					},
				},
			}
			p.EXPECT().Get(reqWithNilFields.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUserConnection(ctx, reqWithNilFields.Req).Return(sampleResponse, nil)
			res, err := act.PluginDeleteUserConnection(ctx, reqWithNilFields)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})
	})
})
