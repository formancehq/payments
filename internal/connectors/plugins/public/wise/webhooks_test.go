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
		ctrl *gomock.Controller
		plg  models.Plugin
		m    *client.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		p := &Plugin{client: m}
		p.supportedWebhooks = map[string]supportedWebhook{
			"test": {
				triggerOn: "transfers#state-change",
				urlPath:   "/transferstatechanged",
				fn:        p.translateTransferStateChangedWebhook,
				version:   "1.0.0",
			},
		}
		plg = p
	})

	AfterEach(func() {
		ctrl.Finish()
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
			expectedWebhookResponseID = "sampleResID"
			webhookBaseUrl = "http://example.com"
			expectedWebhookPath, err = url.JoinPath(webhookBaseUrl, "/transferstatechanged")
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
				"transfers#state-change",
				expectedWebhookPath,
				"1.0.0",
			).Return(
				&client.WebhookSubscriptionResponse{ID: expectedWebhookResponseID},
				nil,
			)

			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Others).To(HaveLen(1))
			Expect(res.Others[0].ID).To(Equal(expectedWebhookResponseID))
		})
	})
})
