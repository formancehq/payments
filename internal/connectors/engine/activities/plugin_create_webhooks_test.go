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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plugin Create Webhooks", func() {
	var (
		act            activities.Activities
		p              *plugins.MockPlugins
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.CreateWebhooksResponse
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.CreateWebhooksResponse{Others: make([]models.PSPOther, 0)}
	})

	Context("plugin create webhook", func() {
		var (
			plugin *models.MockPlugin
			req    activities.CreateWebhooksRequest
			logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
			delay  = 50 * time.Millisecond
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = plugins.NewMockPlugins(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			req = activities.CreateWebhooksRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateWebhooks(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginCreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Others).To(Equal(sampleResponse.Others))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateWebhooks(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginCreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateWebhooks(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginCreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})
	})
})
