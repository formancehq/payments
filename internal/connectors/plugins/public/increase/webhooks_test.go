package increase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Increase Plugin Webhooks", func() {
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
			samplePaymentCreated      *client.Transaction
			samplePayoutCreated       *client.PayoutResponse
			now                       time.Time
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			httpMock = client.NewMockHTTPClient(ctrl)
			verifierMock = NewMockWebhookVerifier(ctrl)
			plg = &Plugin{
				client:   client.New("test", "aseplye", "https://test.com", "we5432345"),
				verifier: verifierMock,
			}
			plg.client.SetHttpClient(httpMock)

			expectedObjectedID = "44"
			expectedWebhookResponseID = "sampleResID"
			webhookBaseURL = "https://example.com"
			now = time.Now().UTC()

			samplePaymentCreated = &client.Transaction{
				ID:        "2",
				AccountID: "2345433",
				Amount:    100,
				CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
				Date:      now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
				Currency:  "USD",
				Source: client.Source{
					TransferID: "123456",
				},
			}
			samplePayoutCreated = &client.PayoutResponse{
				ID:        "4",
				AccountID: "123454",
				Currency:  "USD",
				CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
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
				FromPayload:    json.RawMessage(`{"id":"1", "selected_event_category":"account.created"}`),
				WebhookBaseUrl: webhookBaseURL,
			}

			url, _ := url.JoinPath(req.WebhookBaseUrl, "account/created")

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.EventSubscription{
				ID:                    expectedWebhookResponseID,
				URL:                   url,
				Status:                "active",
				SelectedEventCategory: "account.created",
			})

			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Others).To(HaveLen(1))
			Expect(res.Others[0].ID).To(Equal(expectedWebhookResponseID))
		})

		It("should return an error - create event subscription error", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload:    json.RawMessage(`{"id":"1", "selected_event_category":"account.created"}`),
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
			Expect(err).To(MatchError("failed to create web hooks: test error : : status code: 0"))
			Expect(res).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return an error - check_transfer.updated error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "check_transfer.updated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryCheckTransferUpdated: {
					urlPath: "/check_transfer/updated",
					fn:      plg.translateCheckTransfer,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get check transfer payout: test error : : status code: 0"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - non-https webhook url", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload:    json.RawMessage(`{"id":"1", "selected_event_category":"account.created"}`),
				WebhookBaseUrl: "http://example.com",
			}

			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("webhook URL must use HTTPS protocol"))
			Expect(res).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return an error - invalid payload error", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				FromPayload:    json.RawMessage(`"id":"1", "selected_event_category":"account.created"}`),
				WebhookBaseUrl: webhookBaseURL,
			}

			res, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("invalid character ':' after top-level value"))
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
				Name: "transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
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
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("unknown webhook name"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("translate webhooks - check_transfer.updated", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "check_transfer.updated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryCheckTransferUpdated: {
					urlPath: "/check_transfer/updated",
					fn:      plg.translateCheckTransfer,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CheckPayoutResponse{
				ID:        "4",
				AccountID: "123454",
				Currency:  "USD",
				CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			})

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePayoutCreated.ID))
		})

		It("translate webhooks - check_transfer.updated non-empty DestinationAccountReference", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "check_transfer.updated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryCheckTransferUpdated: {
					urlPath: "/check_transfer/updated",
					fn:      plg.translateCheckTransfer,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CheckPayoutResponse{
				ID:        "4",
				AccountID: "123454",
				Currency:  "USD",
				CreatedAt: now.Add(-time.Duration(50) * time.Minute).UTC().Format(time.RFC3339),
			})

			unknown := "Unknown"
			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses[0].Payment.DestinationAccountReference).To(Equal(&unknown))
		})

		It("should return an error - transaction.created error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get transaction: test error : : status code: 0"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - pending_transaction.created error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "pending_transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryPendingTransactionCreated: {
					urlPath: "/pending_transaction/created",
					fn:      plg.translatePendingTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get pending transaction: test error : : status code: 0"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - pending_transaction.updated error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "pending_transaction.updated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryPendingTransactionUpdated: {
					urlPath: "/pending_transaction/updated",
					fn:      plg.translatePendingTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get pending transaction: test error : : status code: 0"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - declined_transaction.created error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "declined_transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryDeclinedTransactionCreated: {
					urlPath: "/declined_transaction/created",
					fn:      plg.translateDeclinedTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get declined transaction: test error : : status code: 0"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("translate webhooks - pending_transaction.updated", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "pending_transaction.updated",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryPendingTransactionUpdated: {
					urlPath: "/pending_transaction/updated",
					fn:      plg.translatePendingTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, *samplePaymentCreated)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.Source.TransferID))
		})

		It("translate webhooks - pending_transaction.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "pending_transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryPendingTransactionCreated: {
					urlPath: "/pending_transaction/created",
					fn:      plg.translatePendingTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, *samplePaymentCreated)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.Source.TransferID))
		})

		It("translate webhooks - transaction.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, *samplePaymentCreated)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.Source.TransferID))
		})

		It("translate webhooks - declined_transaction.created", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "declined_transaction.created",
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{
						"Increase-Webhook-Signature": {"t=2022-01-31T23:59:59Z,v1=7ebfbadaa1856b9f1374f3e08453de3d760838344862344a103c28129d9173d1"},
					},
					Body: json.RawMessage(fmt.Sprintf(`{"id":"1", "associated_object_id": "%s"}`, expectedObjectedID)),
				},
			}
			plg.webhookConfigs = map[client.EventCategory]webhookConfig{
				client.EventCategoryDeclinedTransactionCreated: {
					urlPath: "/declined_transaction/created",
					fn:      plg.translateDeclinedTransaction,
				},
			}

			verifierMock.EXPECT().verifyWebhookSignature(
				req.Webhook.Body,
				req.Webhook.Headers["Increase-Webhook-Signature"][0],
			).Return(nil)

			httpMock.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, *samplePaymentCreated)

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.Source.TransferID))
		})
	})
})
