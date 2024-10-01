package activities_test

import (
	"fmt"
	"net/http"

	"github.com/formancehq/go-libs/errorsutils"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Plugin Fetch Next Payments", func() {
	var (
		act            activities.Activities
		p              *plugins.MockPlugins
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.FetchNextPaymentsResponse
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.FetchNextPaymentsResponse{HasMore: true}
	})

	Context("plugin fetch next payments", func() {
		var (
			plugin *models.MockPlugin
			req    activities.FetchNextPaymentsRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = plugins.NewMockPlugins(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(s, evts, p)
			req = activities.FetchNextPaymentsRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().FetchNextPayments(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginFetchNextPayments(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.HasMore).To(Equal(sampleResponse.HasMore))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().FetchNextPayments(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginFetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeFalse())
			Expect(temporalErr.Type()).To(Equal(req.ConnectorID.Provider))
		})

		It("returns a non-retryable temporal error", func(ctx SpecContext) {
			wrappedErr := fmt.Errorf("some string: %w", httpwrapper.ErrStatusCodeClientError)
			newErr := errorsutils.NewErrorWithExitCode(wrappedErr, http.StatusTeapot)

			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().FetchNextPayments(ctx, req.Req).Return(sampleResponse, newErr)
			_, err := act.PluginFetchNextPayments(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(req.ConnectorID.Provider))
		})
	})
})
