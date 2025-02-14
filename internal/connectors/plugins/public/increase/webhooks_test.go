package increase

import (
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Increase Plugin Webhooks", func() {
	var (
		plg *Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create webhooks", func() {
		var (
			expectedProfileID         uint64
			expectedWebhookResponseID string
			webhookBaseUrl            string
			err                       error
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			expectedProfileID = 44
			expectedWebhookResponseID = "sampleResID"
			webhookBaseUrl = "http://example.com"
			Expect(err).To(BeNil())
		})

		It("skips making calls when webhook url missing", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"id":%d}`, expectedProfileID)),
			}

			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(client.ErrWebhookUrlMissing))
		})

		It("skips making calls when fromPayload is missing", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}

			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(models.ErrMissingFromPayloadInRequest))
		})

		It("creates webhooks with configured urls", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload:    json.RawMessage(fmt.Sprintf(`{"id":%d}`, expectedProfileID)),
				WebhookBaseUrl: webhookBaseUrl,
			}
			esReq := &client.CreateEventSubscriptionRequest{}
			m.EXPECT().CreateEventSubscription(
				gomock.Any(),
				esReq,
			).Return(
				&client.EventSubscription{ID: expectedWebhookResponseID, URL: webhookBaseUrl},
				nil,
			)

			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Others).To(HaveLen(1))
			Expect(res.Others[0].ID).To(Equal(expectedWebhookResponseID))
		})
	})
})
