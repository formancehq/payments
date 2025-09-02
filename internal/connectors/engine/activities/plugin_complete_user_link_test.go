package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors"
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

var _ = Describe("Plugin Complete User Link", func() {
	var (
		act            activities.Activities
		p              *connectors.MockManager
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.CompleteUserLinkResponse
		logger         = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay          = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.CompleteUserLinkResponse{}
	})

	Context("plugin complete user link", func() {
		var (
			plugin *models.MockPlugin
			req    activities.CompleteUserLinkRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			req = activities.CompleteUserLinkRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
				Req: models.CompleteUserLinkRequest{
					HTTPCallInformation: models.HTTPCallInformation{
						QueryValues: map[string][]string{"code": {"test_code"}},
						Headers:     map[string][]string{"authorization": {"Bearer token"}},
						Body:        []byte("test body"),
					},
					RelatedAttempt: &models.PSUOpenBankingConnectionAttempt{
						ID: uuid.New(),
					},
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CompleteUserLink(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginCompleteUserLink(ctx, req)
			Expect(err).To(BeNil())
			Expect(res).To(Equal(&sampleResponse))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CompleteUserLink(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginCompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error for invalid client request", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CompleteUserLink(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrInvalidClientRequest))
			_, err := act.PluginCompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeInvalidArgument))
		})

		It("returns a non-retryable temporal error for not implemented", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CompleteUserLink(ctx, req.Req).Return(sampleResponse, fmt.Errorf("not implemented: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginCompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})

		It("returns error when plugin not found", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(nil, fmt.Errorf("plugin not found"))
			_, err := act.PluginCompleteUserLink(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})
	})
})
