package activities_test

import (
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Plugin Reverse Transfer", func() {
	var (
		act            activities.Activities
		p              *plugins.MockPlugins
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.ReverseTransferResponse
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.ReverseTransferResponse{
			Payment: models.PSPPayment{Reference: "ref"},
		}
	})

	Context("plugin reverse transfer", func() {
		var (
			plugin *models.MockPlugin
			req    activities.ReverseTransferRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = plugins.NewMockPlugins(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(nil, s, evts, p)
			req = activities.ReverseTransferRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().ReverseTransfer(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginReverseTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Payment.Reference).To(Equal(sampleResponse.Payment.Reference))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().ReverseTransfer(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginReverseTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		It("returns a non-retryable temporal error", func(ctx SpecContext) {
			newErr := status.Errorf(codes.Unimplemented, "invalid")

			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().ReverseTransfer(ctx, req.Req).Return(sampleResponse, newErr)
			_, err := act.PluginReverseTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeUnimplemented))
		})
	})
})
