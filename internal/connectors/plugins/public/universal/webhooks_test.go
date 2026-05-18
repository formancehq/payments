package universal_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal"
	"github.com/formancehq/payments/internal/connectors/plugins/public/universal/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Universal *Plugin — webhooks", func() {
	var (
		ctrl   *gomock.Controller
		mc     *client.MockClient
		plg    *universal.Plugin
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		secret = "test-secret"
		cfg    = json.RawMessage(`{"endpoint":"https://x","apiKey":"k","webhookSharedSecret":"test-secret"}`)
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mc = client.NewMockClient(ctrl)
		mc.EXPECT().SetIdempotencyHeader(gomock.Any()).AnyTimes()
		var err error
		plg, err = universal.New("universal-test", logger, cfg)
		Expect(err).To(BeNil())
		universal.InjectClient(plg, mc)
		universal.InjectDeclared(plg, []models.Capability{models.CAPABILITY_CREATE_WEBHOOKS, models.CAPABILITY_TRANSLATE_WEBHOOKS})
		universal.InjectFeatures(plg, client.Features{WebhookSignature: "hmac-sha256"})
	})

	AfterEach(func() { ctrl.Finish() })

	Context("CreateWebhooks", func() {
		It("registers one subscription per supported event and stashes subscription IDs in metadata", func(ctx SpecContext) {
			mc.EXPECT().CreateWebhookSubscription(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_, _ any, req *client.WebhookSubscriptionRequest) (*client.WebhookSubscriptionResponse, error) {
					return &client.WebhookSubscriptionResponse{ID: "sub_" + req.Name, Name: req.Name}, nil
				}).AnyTimes()

			res, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{
				ConnectorID:    "conn-1",
				WebhookBaseUrl: "https://payments.example.com/webhooks/conn-1",
			})
			Expect(err).To(BeNil())
			Expect(res.Configs).NotTo(BeEmpty())
			Expect(res.Others).To(HaveLen(len(res.Configs)))
			for _, c := range res.Configs {
				Expect(c.Metadata).NotTo(BeNil(), "Metadata must carry the subscription ID for Uninstall")
				Expect(c.Metadata["com.universal.spec/subscription_id"]).To(Equal("sub_" + c.Name))
			}
		})

		It("rejects HTTP base URL on a public hostname", func(ctx SpecContext) {
			_, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{
				ConnectorID:    "conn-1",
				WebhookBaseUrl: "http://insecure.example.com/webhooks",
			})
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("HTTPS"))
		})

		DescribeTable("accepts HTTP only for unambiguously local hostnames",
			func(ctx SpecContext, url string, mustReject bool) {
				mc.EXPECT().CreateWebhookSubscription(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_, _ any, req *client.WebhookSubscriptionRequest) (*client.WebhookSubscriptionResponse, error) {
						return &client.WebhookSubscriptionResponse{ID: "sub_" + req.Name, Name: req.Name}, nil
					}).AnyTimes()

				_, err := plg.CreateWebhooks(ctx, models.CreateWebhooksRequest{
					ConnectorID:    "conn-1",
					WebhookBaseUrl: url,
				})
				if mustReject {
					Expect(err).NotTo(BeNil(), "expected rejection for %s", url)
				} else {
					Expect(err).To(BeNil(), "expected acceptance for %s", url)
				}
			},
			Entry("https on public host", "https://payments.example.com/webhooks", false),
			Entry("http on localhost", "http://localhost:8080/webhooks", false),
			Entry("http on 127.0.0.1", "http://127.0.0.1:8080/webhooks", false),
			Entry("http on docker-internal service name", "http://payments:8080/webhooks", false),
			Entry("http on .local", "http://payments.local/webhooks", false),
			Entry("http on .localhost", "http://payments.localhost/webhooks", false),
			Entry("http on public hostname rejected", "http://insecure.example.com/webhooks", true),
			Entry("ftp scheme rejected", "ftp://payments.example.com/webhooks", true),
		)
	})

	Context("VerifyWebhook", func() {
		It("rejects when signature header is missing", func(ctx SpecContext) {
			_, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{},
				Body:    []byte(`{}`),
			}})
			Expect(err).NotTo(BeNil())
		})

		It("accepts a valid signature and surfaces idempotency key", func(ctx SpecContext) {
			body := []byte(`{"id":"evt-99","type":"payment.updated","createdAt":"2026-05-13T12:00:00Z","resource":{}}`)
			ts := time.Now().UTC().Format(time.RFC3339)
			res, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{
					"X-Universal-Signature": {sign(secret, ts, body)},
					"X-Universal-Timestamp": {ts},
				},
				Body: body,
			}})
			Expect(err).To(BeNil())
			Expect(res.WebhookIdempotencyKey).NotTo(BeNil())
			Expect(*res.WebhookIdempotencyKey).To(Equal("evt-99"))
		})

		It("rejects when signature is wrong", func(ctx SpecContext) {
			body := []byte(`{"id":"evt-99","type":"payment.updated"}`)
			ts := time.Now().UTC().Format(time.RFC3339)
			_, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{
					"X-Universal-Signature": {sign("wrong-secret", ts, body)},
					"X-Universal-Timestamp": {ts},
				},
				Body: body,
			}})
			Expect(err).NotTo(BeNil())
		})

		It("rejects when timestamp is outside tolerance", func(ctx SpecContext) {
			body := []byte(`{"id":"evt-99","type":"payment.updated"}`)
			ts := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
			_, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{
					"X-Universal-Signature": {sign(secret, ts, body)},
					"X-Universal-Timestamp": {ts},
				},
				Body: body,
			}})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrWebhookVerification)).To(BeTrue())
		})

		It("errors when signed body is unparseable (would otherwise lose engine idempotency on retry)", func(ctx SpecContext) {
			body := []byte(`not json`)
			ts := time.Now().UTC().Format(time.RFC3339)
			_, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{
					"X-Universal-Signature": {sign(secret, ts, body)},
					"X-Universal-Timestamp": {ts},
				},
				Body: body,
			}})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrWebhookVerification)).To(BeTrue())
		})

		It("errors when signed body is valid JSON but missing event id", func(ctx SpecContext) {
			body := []byte(`{"type":"payment.updated"}`)
			ts := time.Now().UTC().Format(time.RFC3339)
			_, err := plg.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{
					"X-Universal-Signature": {sign(secret, ts, body)},
					"X-Universal-Timestamp": {ts},
				},
				Body: body,
			}})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrWebhookVerification)).To(BeTrue())
		})
	})

	Context("VerifyWebhook — no configured secret", func() {
		var plgNoSecret *universal.Plugin
		BeforeEach(func() {
			var err error
			plgNoSecret, err = universal.New("u", logger, json.RawMessage(`{"endpoint":"https://x","apiKey":"k"}`))
			Expect(err).To(BeNil())
			universal.InjectClient(plgNoSecret, mc)
		})

		It("accepts unsigned deliveries (counterparty advertised no HMAC)", func(ctx SpecContext) {
			res, err := plgNoSecret.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{},
				Body:    []byte(`{}`),
			}})
			Expect(err).To(BeNil())
			Expect(res.WebhookIdempotencyKey).To(BeNil())
		})

		It("rejects a delivery that carries a signature header (spoof / drift)", func(ctx SpecContext) {
			_, err := plgNoSecret.VerifyWebhook(ctx, models.VerifyWebhookRequest{Webhook: models.PSPWebhook{
				Headers: map[string][]string{
					"X-Universal-Signature": {"deadbeef"},
					"X-Universal-Timestamp": {time.Now().UTC().Format(time.RFC3339)},
				},
				Body: []byte(`{}`),
			}})
			Expect(err).NotTo(BeNil())
			Expect(errors.Is(err, models.ErrWebhookVerification)).To(BeTrue())
		})
	})

	Context("TranslateWebhook", func() {
		It("dispatches payment.updated to a Payment resource", func(ctx SpecContext) {
			body := []byte(`{"id":"e1","type":"payment.updated","createdAt":"2026-05-13T12:00:00Z","resource":{"payment":{"reference":"p1","createdAt":"2026-05-13T12:00:00Z","updatedAt":"2026-05-13T12:00:01Z","type":"PAYIN","status":"SUCCEEDED","amount":"1000","asset":"EUR/2"}}}`)
			res, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{
				Name:    "payment.updated",
				Webhook: models.PSPWebhook{Body: body},
			})
			Expect(err).To(BeNil())
			Expect(res.Responses).To(HaveLen(1))
			Expect(res.Responses[0].Payment).NotTo(BeNil())
			Expect(res.Responses[0].Payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
		})

		It("returns error on unknown event", func(ctx SpecContext) {
			_, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{
				Name:    "made.up",
				Webhook: models.PSPWebhook{Body: []byte(`{}`)},
			})
			Expect(err).NotTo(BeNil())
		})

		DescribeTable("dispatches every event type to its corresponding WebhookResponse field",
			func(ctx SpecContext, name, body string, check func(models.WebhookResponse)) {
				res, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{
					Name:    name,
					Webhook: models.PSPWebhook{Body: []byte(body)},
				})
				Expect(err).To(BeNil())
				Expect(res.Responses).To(HaveLen(1))
				check(res.Responses[0])
			},
			Entry("account.created",
				"account.created",
				`{"id":"e","type":"account.created","createdAt":"2026-05-13T12:00:00Z","resource":{"account":{"reference":"a1","createdAt":"2026-05-13T12:00:00Z"}}}`,
				func(r models.WebhookResponse) { Expect(r.Account).NotTo(BeNil()) },
			),
			Entry("external_account.created",
				"external_account.created",
				`{"id":"e","type":"external_account.created","createdAt":"2026-05-13T12:00:00Z","resource":{"externalAccount":{"reference":"ea","createdAt":"2026-05-13T12:00:00Z"}}}`,
				func(r models.WebhookResponse) { Expect(r.ExternalAccount).NotTo(BeNil()) },
			),
			Entry("balance.updated",
				"balance.updated",
				`{"id":"e","type":"balance.updated","createdAt":"2026-05-13T12:00:00Z","resource":{"balance":{"accountReference":"a1","createdAt":"2026-05-13T12:00:00Z","amount":"100","asset":"EUR/2"}}}`,
				func(r models.WebhookResponse) { Expect(r.Balance).NotTo(BeNil()) },
			),
			Entry("payment.deleted",
				"payment.deleted",
				`{"id":"e","type":"payment.deleted","createdAt":"2026-05-13T12:00:00Z","resource":{"paymentToDelete":"p1"}}`,
				func(r models.WebhookResponse) { Expect(r.PaymentToDelete).NotTo(BeNil()) },
			),
			Entry("payment.cancelled",
				"payment.cancelled",
				`{"id":"e","type":"payment.cancelled","createdAt":"2026-05-13T12:00:00Z","resource":{"paymentToCancel":"p1"}}`,
				func(r models.WebhookResponse) { Expect(r.PaymentToCancel).NotTo(BeNil()) },
			),
		)

		It("rejects order.* and conversion.* webhooks (engine has no surface for them)", func(ctx SpecContext) {
			for _, name := range []string{"order.created", "order.updated", "conversion.created", "conversion.updated"} {
				_, err := plg.TranslateWebhook(ctx, models.TranslateWebhookRequest{
					Name:    name,
					Webhook: models.PSPWebhook{Body: []byte(`{}`)},
				})
				Expect(err).NotTo(BeNil(), "expected %s to be rejected", name)
			}
		})
	})
})

func sign(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
