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
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plugin Delete User", func() {
	var (
		act            activities.Activities
		p              *connectors.MockManager
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.DeleteUserResponse
		logger         = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay          = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.DeleteUserResponse{}
	})

	Context("plugin delete user", func() {
		var (
			plugin *models.MockPlugin
			req    activities.DeleteUserRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			req = activities.DeleteUserRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
				Req: models.DeleteUserRequest{
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
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUser(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginDeleteUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUser(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginDeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error for invalid client request", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUser(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrInvalidClientRequest))
			_, err := act.PluginDeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeInvalidArgument))
		})

		It("returns a non-retryable temporal error for not implemented", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUser(ctx, req.Req).Return(sampleResponse, fmt.Errorf("not implemented: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginDeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})

		It("returns error when plugin not found", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(nil, fmt.Errorf("plugin not found"))
			_, err := act.PluginDeleteUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("handles request with minimal required fields", func(ctx SpecContext) {
			minimalReq := activities.DeleteUserRequest{
				ConnectorID: models.ConnectorID{
					Provider: "minimal_provider",
				},
				Req: models.DeleteUserRequest{
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
				},
			}
			p.EXPECT().Get(minimalReq.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUser(ctx, minimalReq.Req).Return(sampleResponse, nil)
			res, err := act.PluginDeleteUser(ctx, minimalReq)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("handles request with nil optional fields", func(ctx SpecContext) {
			reqWithNilFields := activities.DeleteUserRequest{
				ConnectorID: models.ConnectorID{
					Provider: "nil_fields_provider",
				},
				Req: models.DeleteUserRequest{
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
				},
			}
			p.EXPECT().Get(reqWithNilFields.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().DeleteUser(ctx, reqWithNilFields.Req).Return(sampleResponse, nil)
			res, err := act.PluginDeleteUser(ctx, reqWithNilFields)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})
	})
})
