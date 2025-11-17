package stripe

import (
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin Webhooks", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m, logger: logging.Testing()}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Create Webhooks", func() {
		It("returns client errors", func(ctx SpecContext) {
			expectedErr := errors.New("webhook err")
			req := models.CreateWebhooksRequest{WebhookBaseUrl: "http://example.com"}
			m.EXPECT().CreateWebhookEndpoints(gomock.Any(), req.WebhookBaseUrl).Return(nil, expectedErr)
			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).NotTo(BeNil())
			Expect(err).To(Equal(expectedErr))
		})

		It("returns list of webhooks created", func(ctx SpecContext) {
			rootAccountID := "rooootAcc"
			endpoints := []*stripe.WebhookEndpoint{
				{
					ID:            "id1",
					URL:           "http://example.com/endpoint1",
					Secret:        "seeeecreeet",
					EnabledEvents: []string{"some.event"},
				},
				{
					ID:            "id2",
					URL:           "http://example.com/connect_endpoint2",
					Secret:        "seeeecreeet2",
					EnabledEvents: []string{"some.event", "some.event2"},
				},
			}
			req := models.CreateWebhooksRequest{WebhookBaseUrl: "http://example.com"}
			m.EXPECT().GetRootAccountID().MaxTimes(1).Return(rootAccountID)
			m.EXPECT().CreateWebhookEndpoints(gomock.Any(), req.WebhookBaseUrl).Return(endpoints, nil)
			result, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(result.Others).To(HaveLen(len(endpoints)))
			Expect(result.Configs).To(HaveLen(len(endpoints)))

			configs := result.Configs
			Expect(configs[0].Name).To(Equal("id1"))
			Expect(configs[0].URLPath).To(Equal("/endpoint1"))
			Expect(configs[0].Metadata).To(Equal(map[string]string{
				"secret":                   endpoints[0].Secret,
				webhookRelatedAccountIDKey: rootAccountID,
				"enabled_events":           "some.event",
			}))
			Expect(configs[1].Name).To(Equal("id2"))
			Expect(configs[1].URLPath).To(Equal("/connect_endpoint2"))
			Expect(configs[1].Metadata).To(Equal(map[string]string{
				"secret":         endpoints[1].Secret,
				"enabled_events": "some.event,some.event2",
			}))

			Expect(result.Others[0].ID).To(Equal("id1"))
			Expect(result.Others[1].ID).To(Equal("id2"))
		})
	})
})
