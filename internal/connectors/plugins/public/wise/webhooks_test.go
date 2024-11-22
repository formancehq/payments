package wise

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/plugins/public/wise/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wise Plugin Webhooks", func() {
	var (
		plg *Plugin
		m   *client.MockClient
	)

	BeforeEach(func() {
		plg = &Plugin{}

		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg.SetClient(m)
	})

	Context("create webhooks", func() {
		var (
			expectedProfileID         uint64
			expectedWebhookPath       string
			expectedWebhookResponseID string
			webhookBaseUrl            string
			err                       error
		)

		BeforeEach(func() {
			expectedProfileID = 44
			plg.webhookConfigs = map[string]webhookConfig{
				"test": {
					triggerOn: "transfers#state-change",
					urlPath:   "/transferstatechanged",
					fn:        plg.translateTransferStateChangedWebhook,
					version:   "1.0.0",
				},
			}
			expectedWebhookResponseID = "sampleResID"
			webhookBaseUrl = "http://example.com"
			expectedWebhookPath, err = url.JoinPath(webhookBaseUrl, plg.webhookConfigs["test"].urlPath)
			Expect(err).To(BeNil())
		})

		It("skips making calls when webhook url missing", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload: json.RawMessage(fmt.Sprintf(`{"id":%d}`, expectedProfileID)),
			}

			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError(ErrStackPublicUrlMissing))
		})

		It("creates webhooks with configured urls", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload:    json.RawMessage(fmt.Sprintf(`{"id":%d}`, expectedProfileID)),
				WebhookBaseUrl: webhookBaseUrl,
			}
			m.EXPECT().CreateWebhook(
				gomock.Any(),
				expectedProfileID,
				"test",
				plg.webhookConfigs["test"].triggerOn,
				expectedWebhookPath,
				plg.webhookConfigs["test"].version,
			).Return(
				&client.WebhookSubscriptionResponse{ID: expectedWebhookResponseID},
				nil,
			)

			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Others).To(HaveLen(len(plg.webhookConfigs)))
			Expect(res.Others[0].ID).To(Equal(expectedWebhookResponseID))
		})
	})
})
