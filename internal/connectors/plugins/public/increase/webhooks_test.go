package increase

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/golang/mock/gomock"
)

var _ = Describe("Increase Plugin Webhooks", func() {
	var (
		plg      *Plugin
		httpMock *client.MockHTTPClient
		ctrl     *gomock.Controller
		body     json.RawMessage
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		httpMock = client.NewMockHTTPClient(ctrl)
		plg = &Plugin{
			client:              client.New("test", "aseplye", "https://test.com", "we5432345"),
			webhookSharedSecret: "secret",
		}
		plg.client.SetHttpClient(httpMock)
		body = json.RawMessage(`{"id":"1", "associated_object_id": "test_id"}`)
	})

	Context("create webhooks", func() {
		var (
			expectedObjectedID        string
			expectedWebhookResponseID string
			webhookBaseURL            string
			err                       error
			samplePaymentCreated      *client.Transaction
			now                       time.Time
		)

		BeforeEach(func() {
			plg = &Plugin{
				client: client.New("test", "aseplye", "https://test.com", "we5432345"),
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
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
			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Headers: map[string][]string{},
				},
			}
			_, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(MatchError(client.ErrWebhookHeaderXSignatureMissing))
		})

		It("should return an error - verify signature error", func(ctx SpecContext) {
			timestamp := time.Now().UTC().Format(time.RFC3339)
			signedPayload := fmt.Sprintf("%s.%s", timestamp, string(body))
			expectedSignature, err := computeHMACSHA256(signedPayload, "wrong_secret")
			Expect(err).To(BeNil())

			req := models.VerifyWebhookRequest{
				Webhook: models.PSPWebhook{
					Body: body,
					Headers: map[string][]string{
						HeadersSignature: {fmt.Sprintf("t=%s,v1=%s", timestamp, expectedSignature)},
					},
				},
			}
			_, err = plg.VerifyWebhook(ctx, req)
			Expect(err).To(MatchError("invalid webhook signature: webhook verification error"))
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
			}

			res, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("unknown webhook name"))
			Expect(res).To(Equal(models.TranslateWebhookResponse{}))
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryPendingTransactionCreated: {
					urlPath: "/pending_transaction/created",
					fn:      plg.translatePendingTransaction,
				},
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryPendingTransactionUpdated: {
					urlPath: "/pending_transaction/updated",
					fn:      plg.translatePendingTransaction,
				},
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryDeclinedTransactionCreated: {
					urlPath: "/declined_transaction/created",
					fn:      plg.translateDeclinedTransaction,
				},
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryPendingTransactionUpdated: {
					urlPath: "/pending_transaction/updated",
					fn:      plg.translatePendingTransaction,
				},
			}

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
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
			Expect(res.Responses[0].Payment.ParentReference).To(Equal(samplePaymentCreated.Source.TransferID))
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryPendingTransactionCreated: {
					urlPath: "/pending_transaction/created",
					fn:      plg.translatePendingTransaction,
				},
			}

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
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
			Expect(res.Responses[0].Payment.ParentReference).To(Equal(samplePaymentCreated.Source.TransferID))
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryTransactionCreated: {
					urlPath: "/transaction/created",
					fn:      plg.translateTransaction,
				},
			}

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
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
			Expect(res.Responses[0].Payment.ParentReference).To(Equal(samplePaymentCreated.Source.TransferID))
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
			plg.supportedWebhooks = map[client.EventCategory]supportedWebhook{
				client.EventCategoryDeclinedTransactionCreated: {
					urlPath: "/declined_transaction/created",
					fn:      plg.translateDeclinedTransaction,
				},
			}

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
			Expect(res.Responses[0].Payment.Reference).To(Equal(samplePaymentCreated.ID))
			Expect(res.Responses[0].Payment.ParentReference).To(Equal(samplePaymentCreated.Source.TransferID))
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
