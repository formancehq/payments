package wise

import (
	"fmt"

	"github.com/formancehq/payments/pkg/connectors/wise/client"
	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Wise Plugin Uninstall", func() {
	var (
		ctrl *gomock.Controller
		plg  connector.Plugin
		m    *client.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("uninstall", func() {
		var (
			profiles           []client.Profile
			expectedWebhookID  = "webhook1"
			expectedWebhookID2 = "webhook2"
		)

		BeforeEach(func() {
			profiles = []client.Profile{
				{ID: 1, Type: "type1"},
				{ID: 2, Type: "type2"},
			}
		})

		It("deletes webhooks related to accounts", func(ctx SpecContext) {
			req := connector.UninstallRequest{ConnectorID: "dummyID"}
			m.EXPECT().GetProfiles(gomock.Any()).Return(
				profiles,
				nil,
			)
			m.EXPECT().ListWebhooksSubscription(gomock.Any(), profiles[0].ID).Return(
				[]client.WebhookSubscriptionResponse{
					{ID: expectedWebhookID, Delivery: client.WebhookDelivery{
						URL: fmt.Sprintf("http://somesite.fr/%s", req.ConnectorID),
					}},
					{ID: "skipped", Delivery: client.WebhookDelivery{URL: "http://somesite.fr"}},
				},
				nil,
			)
			m.EXPECT().ListWebhooksSubscription(gomock.Any(), profiles[1].ID).Return(
				[]client.WebhookSubscriptionResponse{
					{ID: expectedWebhookID2, Delivery: client.WebhookDelivery{
						URL: fmt.Sprintf("http://%s.somesite.com", req.ConnectorID),
					}},
				},
				nil,
			)
			m.EXPECT().DeleteWebhooks(gomock.Any(), profiles[0].ID, expectedWebhookID).Return(nil)
			m.EXPECT().DeleteWebhooks(gomock.Any(), profiles[1].ID, expectedWebhookID2).Return(nil)

			_, err := plg.Uninstall(ctx, req)
			Expect(err).To(BeNil())
		})
	})
})
