package activities_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors"
	pluginsError "github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "github.com/golang/mock/gomock"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Activities Suite")
}

var _ = Describe("Plugin Create Bank Account", func() {
	var (
		act            activities.Activities
		p              *connectors.MockManager
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
			logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
			delay  = 50 * time.Millisecond
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			plugin = models.NewMockPlugin(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
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
			Expect(temporalErr.Type()).To(Equal(activities.ErrTypeDefault))
		})

		DescribeTable("converts various errors into non-retryable temporal errors when required",
			func(ctx SpecContext, incomingErr error) {
				p.EXPECT().Get(req.ConnectorID).Return(plugin, nil)
				plugin.EXPECT().CreateBankAccount(ctx, req.Req).Return(sampleResponse, incomingErr)
				_, err := act.PluginCreateBankAccount(ctx, req)
				Expect(err).ToNot(BeNil())
				temporalErr, ok := err.(*temporal.ApplicationError)
				Expect(ok).To(BeTrue())
				Expect(temporalErr.NonRetryable()).To(BeTrue())
				Expect(temporalErr.Type()).To(Equal(activities.ErrTypeInvalidArgument))
			},
			Entry("wrapped invalid client request", fmt.Errorf("invalid: %w", pluginsError.ErrInvalidClientRequest)),
			Entry("connector metadata error custom type", models.NewConnectorValidationError("some-field", models.ErrMissingConnectorMetadata)),
		)
	})
})
