package adyen

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/adyen/adyen-go-api-library/v7/src/webhook"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/adyen/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Adyen Plugin Accounts", func() {
	var (
		plg models.Plugin
		m   *client.MockClient
		now time.Time
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		p := &Plugin{client: m}
		p.initWebhookConfig()
		plg = p
		now = time.Now().UTC()
	})

	Context("creating webhooks", func() {
		It("should fail - stack public url not set", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID: "test",
			}

			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(MatchError("STACK_PUBLIC_URL is not set"))
		})

		It("should fail - wrong url format", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "test",
				WebhookBaseUrl: "&grjete%",
			}

			_, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
		})

		It("should work perfectly", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "test",
				WebhookBaseUrl: "http://localhost:8080/test",
			}

			expectedURL := "http://localhost:8080/test/standard"
			m.EXPECT().CreateWebhook(gomock.Any(), expectedURL, req.ConnectorID).Return(nil)

			configs, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(configs.Configs).To(HaveLen(1))
		})
	})

	Context("verifying webhooks", func() {
		var (
			w webhook.Webhook
		)

		BeforeEach(func() {
			w = webhook.Webhook{
				Live: "false",
				NotificationItems: &[]webhook.NotificationItem{
					{
						NotificationRequestItem: webhook.NotificationRequestItem{
							PspReference: "test",
							Amount: webhook.Amount{
								Currency: "EUR",
								Value:    100,
							},
							EventCode:           webhook.EventCodeAuthorisation,
							EventDate:           &now,
							MerchantAccountCode: "test",
							Operations:          []string{},
							PaymentMethod:       "visa",
							Success:             "true",
						},
					},
				},
			}
		})

		It("should fail - wrong basic auth", func(ctx SpecContext) {
			req := models.VerifyWebhookRequest{
				Config: &models.WebhookConfig{
					Name: "standard",
				},
				Webhook: models.PSPWebhook{
					BasicAuth: &models.BasicAuth{
						Username: "test",
						Password: "test",
					},
					QueryValues: map[string][]string{},
					Headers:     map[string][]string{},
					Body:        []byte{},
				},
			}

			m.EXPECT().VerifyWebhookBasicAuth(req.Webhook.BasicAuth).Return(
				false,
			)

			_, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("invalid basic auth"))
		})

		It("should not process - invalid hmac", func(ctx SpecContext) {
			b, _ := json.Marshal(&w)

			req := models.VerifyWebhookRequest{
				Config: &models.WebhookConfig{
					Name: "standard",
				},
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{},
					Headers:     map[string][]string{},
					Body:        b,
				},
			}

			m.EXPECT().VerifyWebhookBasicAuth(req.Webhook.BasicAuth).Return(true)
			m.EXPECT().TranslateWebhook(string(req.Webhook.Body)).Return(&w, nil)
			m.EXPECT().VerifyWebhookHMAC(gomock.Any()).Return(false)

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(resp.WebhookIdempotencyKey).To(BeEmpty())
		})

		It("should be ok", func(ctx SpecContext) {
			b, _ := json.Marshal(&w)
			ik := sha256.Sum256(b)

			req := models.VerifyWebhookRequest{
				Config: &models.WebhookConfig{
					Name: "standard",
				},
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{},
					Headers:     map[string][]string{},
					Body:        b,
				},
			}

			m.EXPECT().VerifyWebhookBasicAuth(req.Webhook.BasicAuth).Return(true)
			m.EXPECT().TranslateWebhook(string(req.Webhook.Body)).Return(&w, nil)
			m.EXPECT().VerifyWebhookHMAC(gomock.Any()).Return(true)

			resp, err := plg.VerifyWebhook(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.WebhookIdempotencyKey).To(Equal(fmt.Sprintf("%s", string(ik[:]))))
		})
	})

	Context("translating webhooks", func() {

		It("should handle authorization", func(ctx SpecContext) {
			expectedPSPPayment := models.PSPPayment{
				Reference:                   "test",
				CreatedAt:                   now,
				Type:                        models.PAYMENT_TYPE_PAYIN,
				Amount:                      big.NewInt(100),
				Asset:                       "EUR/2",
				Scheme:                      models.PAYMENT_SCHEME_CARD_VISA,
				Status:                      models.PAYMENT_STATUS_AUTHORISATION,
				DestinationAccountReference: pointer.For("test"),
			}

			doTranslateCall(
				ctx,
				plg,
				m,
				webhook.EventCodeAuthorisation,
				100,
				now,
				expectedPSPPayment,
			)
		})
	})

	It("should handle authorisation adjustments", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(150),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_AMOUNT_ADJUSTEMENT,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeAuthorisationAdjustment,
			150,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle cancellation", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(100),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_CARD_VISA,
			Status:                      models.PAYMENT_STATUS_CANCELLED,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeCancellation,
			100,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle capture", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(50),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_CAPTURE,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeCapture,
			50,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle capture failed", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(50),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_CAPTURE_FAILED,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeCaptureFailed,
			50,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle refund", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(50),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_REFUNDED,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeRefund,
			50,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle refund failed", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(50),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_REFUNDED_FAILURE,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeRefundFailed,
			50,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle refund reversed", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(100),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_REFUND_REVERSED,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeRefundedReversed,
			100,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle refund with data", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:             "test1",
			Reference:                   "test",
			CreatedAt:                   now,
			Type:                        models.PAYMENT_TYPE_PAYIN,
			Amount:                      big.NewInt(100),
			Asset:                       "EUR/2",
			Scheme:                      models.PAYMENT_SCHEME_OTHER,
			Status:                      models.PAYMENT_STATUS_REFUNDED,
			DestinationAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodeRefundWithData,
			100,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle payouts to third party", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			Reference:              "test",
			CreatedAt:              now,
			Type:                   models.PAYMENT_TYPE_PAYOUT,
			Amount:                 big.NewInt(100),
			Asset:                  "EUR/2",
			Scheme:                 models.PAYMENT_SCHEME_OTHER,
			Status:                 models.PAYMENT_STATUS_SUCCEEDED,
			SourceAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodePayoutThirdparty,
			100,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle payouts declined", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:        "test1",
			Reference:              "test",
			CreatedAt:              now,
			Type:                   models.PAYMENT_TYPE_PAYOUT,
			Amount:                 big.NewInt(100),
			Asset:                  "EUR/2",
			Scheme:                 models.PAYMENT_SCHEME_OTHER,
			Status:                 models.PAYMENT_STATUS_FAILED,
			SourceAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodePayoutDecline,
			100,
			now,
			expectedPSPPayment,
		)
	})

	It("should handle payouts expired", func(ctx SpecContext) {
		expectedPSPPayment := models.PSPPayment{
			ParentReference:        "test1",
			Reference:              "test",
			CreatedAt:              now,
			Type:                   models.PAYMENT_TYPE_PAYOUT,
			Amount:                 big.NewInt(100),
			Asset:                  "EUR/2",
			Scheme:                 models.PAYMENT_SCHEME_OTHER,
			Status:                 models.PAYMENT_STATUS_EXPIRED,
			SourceAccountReference: pointer.For("test"),
		}

		doTranslateCall(
			ctx,
			plg,
			m,
			webhook.EventCodePayoutExpire,
			100,
			now,
			expectedPSPPayment,
		)
	})
})

func doTranslateCall(
	ctx context.Context,
	plg models.Plugin,
	m *client.MockClient,
	eventCode string,
	amount int64,
	now time.Time,
	expectedPSPPayment models.PSPPayment,
) {
	w := webhook.Webhook{
		Live: "false",
		NotificationItems: &[]webhook.NotificationItem{
			{
				NotificationRequestItem: webhook.NotificationRequestItem{
					OriginalReference: "test1",
					PspReference:      "test",
					Amount: webhook.Amount{
						Currency: "EUR",
						Value:    amount,
					},
					EventCode:           eventCode,
					EventDate:           &now,
					MerchantReference:   "test",
					MerchantAccountCode: "test",
					PaymentMethod:       "visa",
					Success:             "true",
				},
			},
		},
	}

	b, _ := json.Marshal(&w)

	req := models.TranslateWebhookRequest{
		Name: "standard",
		Webhook: models.PSPWebhook{
			QueryValues: map[string][]string{},
			Headers:     map[string][]string{},
			Body:        b,
		},
	}

	m.EXPECT().TranslateWebhook(string(req.Webhook.Body)).Return(&w, nil)

	resp, err := plg.TranslateWebhook(ctx, req)
	Expect(err).To(BeNil())
	Expect(len(resp.Responses)).To(Equal(1))
	comparePayments(*resp.Responses[0].Payment, expectedPSPPayment)
}

func comparePayments(a, b models.PSPPayment) {
	Expect(a.ParentReference).To(Equal(b.ParentReference))
	Expect(a.Reference).To(Equal(b.Reference))
	Expect(a.CreatedAt).To(Equal(b.CreatedAt))
	Expect(a.Type).To(Equal(b.Type))
	Expect(a.Amount.String()).To(Equal(b.Amount.String()))
	Expect(a.Asset).To(Equal(b.Asset))
	Expect(a.Scheme).To(Equal(b.Scheme))
	Expect(a.Status).To(Equal(b.Status))

	switch {
	case a.SourceAccountReference != nil && b.SourceAccountReference != nil:
		Expect(*a.SourceAccountReference).To(Equal(*b.SourceAccountReference))
	case a.SourceAccountReference == nil && b.SourceAccountReference == nil:
	default:
		Fail("SourceAccountReference is not equal")
	}

	switch {
	case a.DestinationAccountReference != nil && b.DestinationAccountReference != nil:
		Expect(*a.DestinationAccountReference).To(Equal(*b.DestinationAccountReference))
	case a.DestinationAccountReference == nil && b.DestinationAccountReference == nil:
	default:
		Fail("DestinationAccountReference is not equal")
	}

	Expect(len(a.Metadata)).To(Equal(len(b.Metadata)))
	for k, v := range a.Metadata {
		Expect(b.Metadata[k]).To(Equal(v))
	}
}
