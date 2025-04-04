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
				client.EventCategoryWireTransferOutgoingCompleted: {
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
				client.EventCategoryWireTransferOutgoingCompleted: {
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

		It("translate webhooks - ach.outgoing_transfer.returned", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.returned",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "return123",
							"type":"RETURN",
							"amount": 500,
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789",
							"description": "Returned ACH transfer",
							"currency_code": "USD"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferReturned: {
					urlPath: "/ach/outgoing_transfer/returned",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("return123"))
		})

		It("translate webhooks - swift.outgoing_transfer.cancellation_requested", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.cancellation_requested",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "cancel123",
							"type": "swift.outgoing_transfer.cancellation_requested",
							"created_at": "2023-01-01T00:00:00Z",
							"updated_at": "2023-01-01T00:00:00Z",
							"initiated_at": "2023-01-01T00:00:00Z",
							"account_number_id": "sample-account-number-id",
							"bank_account_id": "sample-bank-account-id",
							"counterparty_id": "sample-counterparty-id",
							"currency_code": "USD",
							"reason": "Customer request"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingCancellationRequested: {
					urlPath: "/swift/outgoing_transfer/cancellation_requested",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("cancel123"))
		})

		It("translate webhooks - realtime.outgoing_transfer.manual_review", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "realtime.outgoing_transfer.manual_review",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "review123",
							"type": "realtime.outgoing_transfer.manual_review",
							"created_at": "2023-01-01T00:00:00Z",
							"updated_at": "2023-01-01T00:00:00Z",
							"account_number_id": "sample-account-number-id",
							"bank_account_id": "sample-bank-account-id",
							"counterparty_id": "sample-counterparty-id",
							"currency_code": "USD",
							"reason": "Suspicious activity"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryRealtimeTransferManualReview: {
					urlPath: "/realtime/outgoing_transfer/manual_review",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("review123"))
		})
		It("translate webhooks - wire.outgoing_transfer.initiated", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.outgoing_transfer.initiated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "init123",
							"type": "wire.outgoing_transfer.initiated",
							"amount": 2000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferInitiated: {
					urlPath: "/wire/outgoing_transfer/initiated",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("init123"))
		})

		It("translate webhooks - wire.incoming_transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.incoming_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "incoming123",
							"type": "wire.incoming_transfer.completed",
							"amount": 5000,
							"currency_code": "USD",
							"bank_account_id": "account789",
							"counterparty_id": "counterparty987"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferIncomingCompleted: {
					urlPath: "/wire/incoming_transfer/completed",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("incoming123"))
		})

		It("translate webhooks - wire.outgoing_transfer.submitted", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.outgoing_transfer.submitted",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "submitted123",
							"type": "wire.outgoing_transfer.submitted",
							"amount": 3000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty654"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferSubmitted: {
					urlPath: "/wire/outgoing_transfer/submitted",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("submitted123"))
		})

		It("translate webhooks - wire.outgoing_transfer.rejected", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.outgoing_transfer.rejected",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "rejected123",
							"type": "wire.outgoing_transfer.rejected",
							"amount": 4000,
							"currency_code": "USD",
							"bank_account_id": "account321",
							"counterparty_id": "counterparty123"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferRejected: {
					urlPath: "/wire/outgoing_transfer/rejected",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("rejected123"))
		})

		It("translate webhooks - wire.outgoing_transfer.manual_review", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "wire.outgoing_transfer.manual_review",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "manualReview123",
							"type": "wire.outgoing_transfer.manual_review",
							"amount": 6000,
							"currency_code": "USD",
							"bank_account_id": "account654",
							"counterparty_id": "counterparty789",
							"reason": "Compliance check"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryWireTransferManualReview: {
					urlPath: "/wire/outgoing_transfer/manual_review",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("manualReview123"))
		})

		It("translate webhooks - ach.outgoing_transfer.initiated", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.initiated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "initiatedACH123",
							"type": "ach.outgoing_transfer.initiated",
							"amount": 7000,
							"currency_code": "USD",
							"bank_account_id": "account987",
							"counterparty_id": "counterparty654"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferInitiated: {
					urlPath: "/ach/outgoing_transfer/initiated",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("initiatedACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.submitted", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.submitted",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "submittedACH123",
							"type": "ach.outgoing_transfer.submitted",
							"amount": 8000,
							"currency_code": "USD",
							"bank_account_id": "account321",
							"counterparty_id": "counterparty987"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferSubmitted: {
					urlPath: "/ach/outgoing_transfer/submitted",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("submittedACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "completedACH123",
							"type": "ach.outgoing_transfer.completed",
							"amount": 9000,
							"currency_code": "USD",
							"bank_account_id": "account654",
							"counterparty_id": "counterparty321"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferCompleted: {
					urlPath: "/ach/outgoing_transfer/completed",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("completedACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.manual_review", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.manual_review",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "manualReviewACH123",
							"type": "ach.outgoing_transfer.manual_review",
							"amount": 10000,
							"currency_code": "USD",
							"bank_account_id": "account987",
							"counterparty_id": "counterparty654",
							"reason": "Fraud detection"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferManualReview: {
					urlPath: "/ach/outgoing_transfer/manual_review",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("manualReviewACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.canceled", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.canceled",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "cancelledACH123",
							"type": "ach.outgoing_transfer.canceled",
							"amount": 11000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferCanceled: {
					urlPath: "/ach/outgoing_transfer/canceled",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("cancelledACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.return_dishonored", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.return_dishonored",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "dishonoredACH123",
							"type": "ach.outgoing_transfer.return_dishonored",
							"amount": 12000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferReturnDishonored: {
					urlPath: "/ach/outgoing_transfer/return_dishonored",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("dishonoredACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.return_contested", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.return_contested",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "contestedACH123",
							"type": "ach.outgoing_transfer.return_contested",
							"amount": 13000,
							"currency_code": "USD",
							"bank_account_id": "account789",
							"counterparty_id": "counterparty123"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferReturnContested: {
					urlPath: "/ach/outgoing_transfer/return_contested",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("contestedACH123"))
		})

		It("translate webhooks - ach.outgoing_transfer.noc", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.outgoing_transfer.noc",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "nocACH123",
							"type": "ach.outgoing_transfer.noc",
							"amount": 14000,
							"currency_code": "USD",
							"bank_account_id": "account321",
							"counterparty_id": "counterparty654"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHTransferNOC: {
					urlPath: "/ach/outgoing_transfer/noc",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("nocACH123"))
		})

		It("translate webhooks - ach.incoming_transfer.scheduled", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.scheduled",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "scheduledACH123",
							"type": "ach.incoming_transfer.scheduled",
							"amount": 15000,
							"currency_code": "USD",
							"bank_account_id": "account654",
							"counterparty_id": "counterparty987"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingScheduled: {
					urlPath: "/ach/incoming_transfer/scheduled",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("scheduledACH123"))
		})

		It("translate webhooks - ach.incoming_transfer.settled", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.settled",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "settledACH123",
							"type": "ach.incoming_transfer.settled",
							"amount": 16000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingSettled: {
					urlPath: "/ach/incoming_transfer/settled",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("settledACH123"))
		})

		It("translate webhooks - ach.incoming_transfer.nsf", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.nsf",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "nsfACH123",
							"type": "ach.incoming_transfer.nsf",
							"amount": 17000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingNSF: {
					urlPath: "/ach/incoming_transfer/nsf",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("nsfACH123"))
		})

		It("translate webhooks - ach.incoming_transfer.completed", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.completed",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "completedACHIncoming123",
							"type": "ach.incoming_transfer.completed",
							"amount": 18000,
							"currency_code": "USD",
							"bank_account_id": "account789",
							"counterparty_id": "counterparty123"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingCompleted: {
					urlPath: "/ach/incoming_transfer/completed",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("completedACHIncoming123"))
		})

		It("translate webhooks - ach.incoming_transfer.returned", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.returned",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "returnedACHIncoming123",
							"type": "ach.incoming_transfer.returned",
							"amount": 19000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingReturned: {
					urlPath: "/ach/incoming_transfer/returned",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("returnedACHIncoming123"))
		})

		It("translate webhooks - ach.incoming_transfer.return_dishonored", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.return_dishonored",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "dishonoredACHIncoming123",
							"type": "ach.incoming_transfer.return_dishonored",
							"amount": 20000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingReturnDishonored: {
					urlPath: "/ach/incoming_transfer/return_dishonored",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("dishonoredACHIncoming123"))
		})

		It("translate webhooks - ach.incoming_transfer.return_contested", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "ach.incoming_transfer.return_contested",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "contestedACHIncoming123",
							"type": "ach.incoming_transfer.return_contested",
							"amount": 21000,
							"currency_code": "USD",
							"bank_account_id": "account789",
							"counterparty_id": "counterparty123"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryACHIncomingReturnContested: {
					urlPath: "/ach/incoming_transfer/return_contested",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("contestedACHIncoming123"))
		})

		It("translate webhooks - swift.outgoing_transfer.initiated", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.initiated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "initiatedSWIFT123",
							"type": "swift.outgoing_transfer.initiated",
							"amount": 22000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingInitiated: {
					urlPath: "/swift/outgoing_transfer/initiated",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("initiatedSWIFT123"))
		})

		It("translate webhooks - swift.outgoing_transfer.manual_review", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.manual_review",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "manualReviewSWIFT123",
							"type": "swift.outgoing_transfer.manual_review",
							"amount": 23000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789",
							"reason": "Compliance check"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingManualReview: {
					urlPath: "/swift/outgoing_transfer/manual_review",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("manualReviewSWIFT123"))
		})

		It("translate webhooks - swift.outgoing_transfer.submitted", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.submitted",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "submittedSWIFT123",
							"type": "swift.outgoing_transfer.submitted",
							"amount": 24000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingSubmitted: {
					urlPath: "/swift/outgoing_transfer/submitted",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("submittedSWIFT123"))
		})

		It("translate webhooks - swift.outgoing_transfer.pending_return", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.pending_return",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "pendingReturnSWIFT123",
							"type": "swift.outgoing_transfer.pending_return",
							"amount": 25000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingPendingReturn: {
					urlPath: "/swift/outgoing_transfer/pending_return",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("pendingReturnSWIFT123"))
		})

		It("translate webhooks - swift.outgoing_transfer.returned", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.returned",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "returnedSWIFT123",
							"type": "swift.outgoing_transfer.returned",
							"amount": 26000,
							"currency_code": "USD",
							"bank_account_id": "account789",
							"counterparty_id": "counterparty123"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingReturned: {
					urlPath: "/swift/outgoing_transfer/returned",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("returnedSWIFT123"))
		})

		It("translate webhooks - swift.outgoing_transfer.cancellation_accepted", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.cancellation_accepted",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "cancellationAcceptedSWIFT123",
							"type": "swift.outgoing_transfer.cancellation_accepted",
							"amount": 27000,
							"currency_code": "USD",
							"bank_account_id": "account123",
							"counterparty_id": "counterparty456"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingCancellationAccepted: {
					urlPath: "/swift/outgoing_transfer/cancellation_accepted",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("cancellationAcceptedSWIFT123"))
		})

		It("translate webhooks - swift.outgoing_transfer.cancellation_rejected", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "swift.outgoing_transfer.cancellation_rejected",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Column-Signature": {"7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(`{
						"id":"1", 
						"data": {
							"id": "cancellationRejectedSWIFT123",
							"type": "swift.outgoing_transfer.cancellation_rejected",
							"amount": 28000,
							"currency_code": "USD",
							"bank_account_id": "account456",
							"counterparty_id": "counterparty789"
						}
					}`),
				},
			}

			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategorySwiftOutgoingCancellationRejected: {
					urlPath: "/swift/outgoing_transfer/cancellation_rejected",
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
			Expect(res.Responses[0].Payment.Reference).To(Equal("cancellationRejectedSWIFT123"))
		})
	})
})
