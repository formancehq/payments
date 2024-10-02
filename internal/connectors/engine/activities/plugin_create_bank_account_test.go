package activities_test

import (
	"fmt"
	"net/http"
	"testing"

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

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Activities Suite")
}

var _ = Describe("Plugin Create Bank Account", func() {
	var (
		act            activities.Activities
		p              *plugins.MockPlugins
		s              *storage.MockStorage
		evts           *events.Events
		sampleResponse models.CreateBankAccountResponse
	)

	BeforeEach(func() {
		evts = &events.Events{}
		sampleResponse = models.CreateBankAccountResponse{
			RelatedAccount: models.PSPAccount{Reference: "ref"},
		}
	})

	Context("plugin create bank account", func() {
		var (
			plugin *models.MockPlugin
			req    activities.CreateBankAccountRequest
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = plugins.NewMockPlugins(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(s, evts, p)
			req = activities.CreateBankAccountRequest{
				ConnectorID: models.ConnectorID{
					Provider: "some_provider",
				},
			}
		})

		It("calls underlying plugin", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateBankAccount(ctx, req.Req).Return(sampleResponse, nil)
			res, err := act.PluginCreateBankAccount(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.RelatedAccount.Reference).To(Equal(sampleResponse.RelatedAccount.Reference))
		})

		It("returns a retryable temporal error", func(ctx SpecContext) {
			p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
			plugin.EXPECT().CreateBankAccount(ctx, req.Req).Return(sampleResponse, fmt.Errorf("some string"))
			_, err := act.PluginCreateBankAccount(ctx, req)
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
			plugin.EXPECT().CreateBankAccount(ctx, req.Req).Return(sampleResponse, newErr)
			_, err := act.PluginCreateBankAccount(ctx, req)
			Expect(err).ToNot(BeNil())
			temporalErr, ok := err.(*temporal.ApplicationError)
			Expect(ok).To(BeTrue())
			Expect(temporalErr.NonRetryable()).To(BeTrue())
			Expect(temporalErr.Type()).To(Equal(req.ConnectorID.Provider))
		})
	})
})
