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

var _ = Describe("Plugin Create User", func() {
	var (
		act            activities.Activities
		p              *connectors.MockManager
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.CreateUserResponse
		logger         = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay          = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.CreateUserResponse{
			PermanentToken: &models.Token{
				Token:     "permanent-token-123",
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			Metadata: map[string]string{
				"user_id": "external-user-123",
				"status":  "active",
			},
		}
	})

	Context("plugin create user", func() {
		var (
			plugin *models.MockPlugin
			req    activities.CreateUserRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			req = activities.CreateUserRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
				Req: models.CreateUserRequest{
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
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginCreateUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginCreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error for invalid client request", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrInvalidClientRequest))
			_, err := act.PluginCreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeInvalidArgument))
		})

		It("returns a non-retryable temporal error for not implemented", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(ctx, req.Req).Return(sampleResponse, fmt.Errorf("not implemented: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginCreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})

		It("returns error when plugin not found", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(nil, fmt.Errorf("plugin not found"))
			_, err := act.PluginCreateUser(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("handles response without permanent token", func(ctx SpecContext) {
			responseWithoutToken := models.CreateUserResponse{
				PermanentToken: nil,
				Metadata: map[string]string{
					"user_id": "external-user-456",
					"status":  "pending",
				},
			}
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(ctx, req.Req).Return(responseWithoutToken, nil)
			res, err := act.PluginCreateUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&responseWithoutToken))
			Expect(res.PermanentToken).To(BeNil())
		})

		It("handles response with empty metadata", func(ctx SpecContext) {
			responseWithEmptyMetadata := models.CreateUserResponse{
				PermanentToken: &models.Token{
					Token:     "token-789",
					ExpiresAt: time.Now().Add(time.Hour),
				},
				Metadata: map[string]string{},
			}
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateUser(ctx, req.Req).Return(responseWithEmptyMetadata, nil)
			res, err := act.PluginCreateUser(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&responseWithEmptyMetadata))
			Expect(res.Metadata).To(BeEmpty())
		})
	})
})
