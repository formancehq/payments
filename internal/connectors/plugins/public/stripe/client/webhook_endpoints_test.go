package client_test

import (
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Client WebhookEndpoints", func() {
	var (
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		cl     client.Client
		ctrl   *gomock.Controller
		b      *client.MockBackend
		token  string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		b = client.NewMockBackend(ctrl)
		token = "dummy"
		cl = client.New("test", logger, b, token)
	})

	Context("Create Webhook Endpoints", func() {
		It("fails when underlying calls fail", func(ctx SpecContext) {
			expectedErr := errors.New("some err")

			b.EXPECT().Call("POST", "/v1/webhook_endpoints", token, gomock.Any(), gomock.Any()).Return(expectedErr)
			_, err := cl.CreateWebhookEndpoints(ctx, "http://example.com")
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns the webhook endpoint", func(ctx SpecContext) {
			someID := "theID"
			expected := &stripe.WebhookEndpoint{}
			b.EXPECT().Call("POST", "/v1/webhook_endpoints", token, gomock.Any(), expected).Times(2).
				DoAndReturn(func(_, _, _ string, _ any, exp *stripe.WebhookEndpoint) error {
					exp.ID = someID
					return nil
				})
			results, err := cl.CreateWebhookEndpoints(ctx, "http://example.com")
			Expect(err).To(BeNil())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ID).To(Equal(someID))
		})
	})
})
