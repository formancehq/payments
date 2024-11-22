package activities_test

import (
	"fmt"

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

var _ = Describe("Plugin Poll Transfer Status", func() {
	var (
		act            activities.Activities
		p              *plugins.MockPlugins
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.PollTransferStatusResponse
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.PollTransferStatusResponse{
			Payment: &models.PSPPayment{Reference: "ref"},
		}
	})

	Context("plugin poll transfer staus", func() {
		var (
			plugin *models.MockPlugin
			req    activities.PollTransferStatusRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = plugins.NewMockPlugins(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(nil, s, evts, p)
			req = activities.PollTransferStatusRequest{
				ConnectorID: models.ConnectorID{Provider: "some_provider"},
				Req: models.PollTransferStatusRequest{
					TransferID: "test",
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().PollTransferStatus(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginPollTransferStatus(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Payment.Reference).To(Equal(sampleResponse.Payment.Reference))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().PollTransferStatus(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginPollTransferStatus(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().PollTransferStatus(ctx, req.Req).Return(sampleResponse, fmt.Errorf("invalid: %w", pluginsError.ErrNotImplemented))
			_, err := act.PluginPollTransferStatus(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})
	})
})
