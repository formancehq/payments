package increase

import (
	"encoding/json"
	"fmt"
	"time"

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
			expectedObjectedID         string
			expectedWebhookResponseID string
			webhookBaseUrl            string
			err                       error
			sampleAccountCreated      *client.Account
			samplePaymentCreated      *client.Transaction
			sampleTransferCreated      *client.TransferResponse
			sampleExternalAccountCreated *client.ExternalAccount
			samplePayoutCreated *client.PayoutResponse
			now            time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			expectedObjectedID = "44"
			expectedWebhookResponseID = "sampleResID"
			webhookBaseUrl = "http://example.com"
			now = time.Now().UTC()

			sampleAccountCreated = &client.Account{
				ID:        "1",
				Name:     "Account 1",
				Currency:  "USD",
				CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			}
			samplePaymentCreated = &client.Transaction{
				ID:        "2",
				AccountID: "2345433",
				Amount:    "100.01",
				CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
				Date:      now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
				Currency:  "USD",
			}
			sampleExternalAccountCreated = &client.ExternalAccount{
				ID:            "4",
				Description:   "Account 1",
				AccountNumber: "123454",
				CreatedAt:     now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			}
			sampleTransferCreated = &client.TransferResponse{
				ID:            "4",
				Description:   "Account 1",
				AccountID: "123454",
				Currency: "USD",
				CreatedAt:     now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			}
			samplePayoutCreated = &client.PayoutResponse{
				ID:            "4",
				AccountID: "123454",
				Currency: "USD",
				CreatedAt:     now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			}
			Expect(err).To(BeNil())
		})

		It("skips making calls when webhook url missing", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload: json.RawMessage(`{"id":"1"}`),
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
				FromPayload:    json.RawMessage(`{"id":"1"}`),
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

		It("should return an error - validation error - no header signature", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
					Body: json.RawMessage(`{"id":"1"}`),
				},
			}

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing X-Signature-Sha256 header"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("translate webhooks - account.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"account.created"}`, expectedObjectedID)),
				},
			}

			m.EXPECT().VerifyWebhookSignature(
				gomock.Any(),
				gomock.Any(),
			).Return(nil)

			m.EXPECT().GetAccount(
				gomock.Any(),
				expectedObjectedID,
			).Return(
				sampleAccountCreated,
				nil,
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(*res.Responses[0].Account.Name).To(Equal(sampleAccountCreated.Name))
		})

		It("translate webhooks - declined_transaction.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"declined_transaction.created"}`, expectedObjectedID)),
				},
			}

			m.EXPECT().VerifyWebhookSignature(
				gomock.Any(),
				gomock.Any(),
			).Return(nil)

			m.EXPECT().GetDeclinedTransaction(
				gomock.Any(),
				expectedObjectedID,
			).Return(
				samplePaymentCreated,
				nil,
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
		})

		It("translate webhooks - pending_transaction.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"pending_transaction.created"}`, expectedObjectedID)),
				},
			}

			m.EXPECT().VerifyWebhookSignature(
				gomock.Any(),
				gomock.Any(),
			).Return(nil)

			m.EXPECT().GetPendingTransaction(
				gomock.Any(),
				expectedObjectedID,
			).Return(
				samplePaymentCreated,
				nil,
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
		})

		It("translate webhooks - transaction.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"transaction.created"}`, expectedObjectedID)),
				},
			}

			m.EXPECT().VerifyWebhookSignature(
				gomock.Any(),
				gomock.Any(),
			).Return(nil)

			m.EXPECT().GetTransaction(
				gomock.Any(),
				expectedObjectedID,
			).Return(
				samplePaymentCreated,
				nil,
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
		})

		It("translate webhooks - external_account.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"external_account.created"}`, expectedObjectedID)),
				},
			}

			m.EXPECT().VerifyWebhookSignature(
				gomock.Any(),
				gomock.Any(),
			).Return(nil)

			m.EXPECT().GetExternalAccount(
				gomock.Any(),
				expectedObjectedID,
			).Return(
				sampleExternalAccountCreated,
				nil,
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Account.Reference).To(Equal(sampleExternalAccountCreated.ID))
		})

		It("translate webhooks - account_transfer.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"account_transfer.created"}`, expectedObjectedID)),
				},
			}
		
			m.EXPECT().VerifyWebhookSignature(gomock.Any(), gomock.Any()).Return(nil)
			m.EXPECT().GetTransfer(gomock.Any(), expectedObjectedID).Return(sampleTransferCreated, nil)
		
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(sampleTransferCreated.ID))
		})

		It("translate webhooks - check_transfer.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"check_transfer.created"}`, expectedObjectedID)),
				},
			}
		
			m.EXPECT().VerifyWebhookSignature(gomock.Any(), gomock.Any()).Return(nil)
			m.EXPECT().GetCheckTransferPayout(gomock.Any(), expectedObjectedID).Return(samplePayoutCreated, nil)
		
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePayoutCreated.ID))
		})
		
		It("translate webhooks - wire_transfer.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"wire_transfer.created"}`, expectedObjectedID)),
				},
			}
		
			m.EXPECT().VerifyWebhookSignature(gomock.Any(), gomock.Any()).Return(nil)
			m.EXPECT().GetWireTransferPayout(gomock.Any(), expectedObjectedID).Return(samplePayoutCreated, nil)
		
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePayoutCreated.ID))
		})
		
		It("translate webhooks - real_time_payments_transfer.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"real_time_payments_transfer.created"}`, expectedObjectedID)),
				},
			}
		
			m.EXPECT().VerifyWebhookSignature(gomock.Any(), gomock.Any()).Return(nil)
			m.EXPECT().GetRTPTransferPayout(gomock.Any(), expectedObjectedID).Return(samplePayoutCreated, nil)
		
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePayoutCreated.ID))
		})
		
		It("translate webhooks - ach_transfer.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"Increase-Webhook-Signature: t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s", "category":"ach_transfer.created"}`, expectedObjectedID)),
				},
			}
		
			m.EXPECT().VerifyWebhookSignature(gomock.Any(), gomock.Any()).Return(nil)
			m.EXPECT().GetACHTransferPayout(gomock.Any(), expectedObjectedID).Return(samplePayoutCreated, nil)
		
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePayoutCreated.ID))
		})
	})
})
