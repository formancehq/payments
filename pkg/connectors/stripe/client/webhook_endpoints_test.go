package client_test

import (
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
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
		err    error
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		b = client.NewMockBackend(ctrl)
		token = "dummy"
		b.EXPECT().Call("GET", "/v1/account", token, nil, &stripe.Account{}).DoAndReturn(
			func(_, _, _ string, _ any, account *stripe.Account) error {
				account.ID = "rootID"
				return nil
			})
		cl, err = client.New("test", logger, b, token)
		Expect(err).To(BeNil())
	})

	Context("Create Webhook Endpoints", func() {
		It("fails when underlying call to check for existing webhooks fails", func(ctx SpecContext) {
			expectedErr := errors.New("some err")

			b.EXPECT().CallRaw("GET", "/v1/webhook_endpoints", token, gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			_, err := cl.CreateWebhookEndpoints(ctx, "http://example.com")
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("fails when underlying calls fail", func(ctx SpecContext) {
			expectedErr := errors.New("some err")

			b.EXPECT().CallRaw("GET", "/v1/webhook_endpoints", token, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			b.EXPECT().Call("POST", "/v1/webhook_endpoints", token, gomock.Any(), gomock.Any()).Return(expectedErr)
			_, err := cl.CreateWebhookEndpoints(ctx, "http://example.com")
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("returns the webhook endpoints", func(ctx SpecContext) {
			someID := "theID"
			expected := &stripe.WebhookEndpoint{}
			b.EXPECT().CallRaw("GET", "/v1/webhook_endpoints", token, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
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

		It("updates existing webhooks instead of creating new ones", func(ctx SpecContext) {
			urlBase := "http://example.com"
			newID := "newID"
			existingID := "existingID"
			enabledEvents := make([]string, 0)

			expected := &stripe.WebhookEndpoint{}
			b.EXPECT().CallRaw("GET", "/v1/webhook_endpoints", token, gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_, _, _ string, _ any, _ any, exp *stripe.WebhookEndpointList) error {
					exp.Data = []*stripe.WebhookEndpoint{
						{
							ID:            existingID,
							URL:           urlBase + "/root",
							EnabledEvents: []string{"some.event"},
						},
					}
					return nil
				})
			b.EXPECT().Call("POST", "/v1/webhook_endpoints/"+existingID, token, gomock.Any(), expected).Times(1).
				DoAndReturn(func(_, _, _ string, params *stripe.WebhookEndpointParams, exp *stripe.WebhookEndpoint) error {
					exp.ID = existingID
					exp.URL = urlBase + "/root"
					for _, evt := range params.EnabledEvents {
						exp.EnabledEvents = append(exp.EnabledEvents, *evt)
						enabledEvents = append(enabledEvents, *evt)
					}
					return nil
				})
			b.EXPECT().Call("POST", "/v1/webhook_endpoints", token, gomock.Any(), expected).Times(1).
				DoAndReturn(func(_, _, _ string, _ any, exp *stripe.WebhookEndpoint) error {
					exp.ID = newID
					return nil
				})
			results, err := cl.CreateWebhookEndpoints(ctx, "http://example.com")
			Expect(err).To(BeNil())
			Expect(results).To(HaveLen(2))
			Expect(results[0].ID).To(Equal(existingID))
			Expect(results[1].ID).To(Equal(newID))
			Expect(results[0].EnabledEvents).To(Equal(enabledEvents))
		})
	})
})
