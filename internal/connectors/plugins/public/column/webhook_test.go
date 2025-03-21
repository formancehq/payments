package column

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Column Plugin Webhooks", func() {
	var (
		plg      *Plugin
		httpMock *client.MockHTTPClient
		ctrl     *gomock.Controller
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create webhooks", func() {
		var (
			expectedObjectedID        string
			expectedWebhookResponseID string
			webhookBaseURL            string
			err                       error
			verifierMock              *MockWebhookVerifier
			secret                    string
		)
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			httpMock = client.NewMockHTTPClient(ctrl)
			verifierMock = NewMockWebhookVerifier(ctrl)
			plg = &Plugin{
				client:   client.New("test", "aseplye", "https://test.com"),
				verifier: verifierMock,
			}
			plg.client.SetHttpClient(httpMock)
			expectedObjectedID = "44"
			expectedWebhookResponseID = "sampleResID"
			webhookBaseURL = "https://example.com"
			secret = "test-secret"
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferSettled: {
					urlPath: "/ach/outgoing_transfer/settled",
					fn:      plg.translateAchTransfer,
				},
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
				WebhookBaseUrl: webhookBaseURL,
			}
			url, _ := url.JoinPath(req.WebhookBaseUrl, "ach/outgoing_transfer/settled")
			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.EventSubscription{
				ID:            expectedWebhookResponseID,
				URL:           url,
				Secret:        "test-secret",
				EnabledEvents: []string{"ach.outgoing_transfer.settled"},
			})
			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Others).To(HaveLen(1))
			Expect(res.Others[0].ID).To(Equal(expectedWebhookResponseID))
		})

		It("should return an error - create event subscription error", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload:    json.RawMessage(`{"id":"1"}`),
				WebhookBaseUrl: webhookBaseURL,
			}
			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)
			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create webhook subscription: failed to create web hooks: test error : "))
			Expect(res).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return an error - validation error - no header signature", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
					Body:    json.RawMessage(`{"id":"1"}`),
				},
			}
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing X-Signature-Sha256 header"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - verify signature error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.settled",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s", "type": "ach.outgoing_transfer.settled"}}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferSettled: {
					urlPath: "/ach/outgoing_transfer/settled",
					fn:      plg.translateAchTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(errors.New("test error"))

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - unknown webhook name error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ac.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s"}}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferSettled: {
					urlPath: "/ach/outgoing_transfer/settled",
					fn:      plg.translateAchTransfer,
				},
			}
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("unknown webhook name"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - ach.outgoing_transfer.settled error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.settled",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s"}}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferSettled: {
					urlPath: "/ach/outgoing_transfer/settled",
					fn:      plg.translateAchTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(errors.New("test error"))

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error book.transfer.completed error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "book.transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s", "type": "book.transfer.completed"}}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryBookTransferCompleted: {
					urlPath: "/book/transfer/completed",
					fn:      plg.translateBookTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(errors.New("test error"))

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error realtime.outgoing_transfer.completed error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "realtime.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s", "type": "realtime.outgoing_transfer.completed"}}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryRealtimeTransferCompleted: {
					urlPath: "/realtime/outgoing_transfer/completed",
					fn:      plg.translateRealtimeTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(errors.New("test error"))

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - swift.outgoing_transfer.completed error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s", "type":"swift.outgoing_transfer.completed"}}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryInternationalWireCompleted: {
					urlPath: "/swift/outgoing_transfer/completed",
					fn:      plg.translateInternationalWireTransfer,
					secret:  secret,
				},
			}
			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(errors.New("test error"))

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - wire.outgoing_transfer.completed error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s", "type": "wire.outgoing_transfer.completed"}}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferCompleted: {
					urlPath: "/wire/outgoing_transfer/completed",
					fn:      plg.translateWireTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(errors.New("test error"))

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("translate webhooks - book.transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "book.transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{
						"id":"1", 
						"data": {
							"id": "%s",
							"type":"book.transfer.completed",
							"created_at": "2023-01-01T00:00:00Z",
							"updated_at": "2023-01-01T00:00:00Z",
							"idempotency_key": "sample-idempotency-key",
							"sender_bank_account_id": "sample-sender-bank-account-id",
							"sender_account_number_id": "sample-sender-account-number-id",
							"receiver_bank_account_id": "sample-receiver-bank-account-id",
							"receiver_account_number_id": "sample-receiver-account-number-id",
							"currency_code": "USD"
						}
					}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryBookTransferCompleted: {
					urlPath: "/book/transfer/completed",
					fn:      plg.translateBookTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(expectedObjectedID))
		})

		It("translate webhooks - realtime.outgoing_transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "realtime.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{
						"id":"1", 
						"data": {
							"id": "%s",
							"accepted_at": "2023-12-29T19:45:11Z",
							"account_number_id": "acno_2XrFelm5efqwGkPsu3B1DtSEDDg",
							"allow_overdraft": false,
							"amount": 10000,
							"bank_account_id": "bacc_2XrFelZxSUOXXTswfr0h9KByzNp",
							"blocked_at": null,
							"completed_at": "2023-12-29T19:45:13Z",
							"counterparty_id": "cpty_2aELmewqaBj5Bp6oraJ7Pl6LH1p",
							"ultimate_debtor_counterparty_id": null,
							"currency_code": "USD",
							"description": "Example realtime transfer"
						}
					}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryRealtimeTransferCompleted: {
					urlPath: "/realtime/outgoing_transfer/completed",
					fn:      plg.translateRealtimeTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(expectedObjectedID))
		})

		It("translate webhooks - ach.outgoing_transfer.settled", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.settled",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "rule456",
							"type":"CREDIT",
							"amount": 1000,
							"bank_account_id": "account789",
							"counterparty_id": "counterparty123",
							"description": "Test description",
							"currency_code": "USD"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferSettled: {
					urlPath: "/ach/outgoing_transfer/settled",
					fn:      plg.translateAchTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal("rule456"))
		})

		It("translate webhooks - swift.outgoing_transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "eodl",
							"type": "swift.outgoing_transfer.completed",
							"created_at": "2023-01-01T00:00:00Z",
							"updated_at": "2023-01-01T00:00:00Z",
							"initiated_at": "2023-01-01T00:00:00Z",
							"pending_submission_at": "2023-01-01T00:00:00Z",
							"submitted_at": "2023-01-01T00:00:00Z",
							"account_number_id": "sample-account-number-id",
							"bank_account_id": "sample-bank-account-id",
							"counterparty_id": "sample-counterparty-id",
							"fx_quote_id": "sample-fx-quote-id",
							"charge_bearer": "SHAR",
							"remittance_info": {
								"reference": "sample-reference",
								"unstructured": "sample-unstructured"
							}
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryInternationalWireCompleted: {
					urlPath: "/swift/outgoing_transfer/completed",
					fn:      plg.translateInternationalWireTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal("eodl"))
		})

		It("translate webhooks - wire.outgoing_transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "data": {"id": "%s", "type": "wire.outgoing_transfer.completed"}}`, expectedObjectedID)),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferCompleted: {
					urlPath: "/wire/outgoing_transfer/completed",
					fn:      plg.translateWireTransfer,
					secret:  secret,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Column-Signature"][0],
				secret,
			).Return(nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(expectedObjectedID))
		})
	})
})
