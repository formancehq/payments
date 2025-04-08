package mangopay

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Create Webhooks", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create webhooks", func() {
		var (
			m                         *client.MockClient
			listAllValidHooksResponse []*client.Hook
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			plg.initWebhookConfig()

			listAllValidHooksResponse = []*client.Hook{
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypeTransferNormalCreated,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypeTransferNormalFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypeTransferNormalSucceeded,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayoutNormalCreated,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayoutNormalFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayoutNormalSucceeded,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayoutInstantFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayoutInstantSucceeded,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayinNormalCreated,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayinNormalSucceeded,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayinNormalFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypeTransferRefundFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypeTransferRefundSucceeded,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayOutRefundFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayOutRefundSucceeded,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayinRefundFailed,
				},
				{
					ID:        "1",
					URL:       "test",
					Validity:  "VALID",
					EventType: client.EventTypePayinRefundSucceeded,
				},
			}
		})

		It("should return an error - missing stack url", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("webhook base URL is required: invalid request"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return an error - get active hooks error", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "test",
				WebhookBaseUrl: "http://localhost:8080",
			}

			m.EXPECT().ListAllHooks(gomock.Any()).Return(nil, errors.New("test error"))

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return an error - update hook error", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "test",
				WebhookBaseUrl: "http://localhost:8080",
			}

			m.EXPECT().ListAllHooks(gomock.Any()).Return(listAllValidHooksResponse, nil)
			m.EXPECT().UpdateHook(gomock.Any(), "1", gomock.Any()).
				Return(errors.New("test error"))

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should return an error - create hook error", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "test",
				WebhookBaseUrl: "http://localhost:8080",
			}

			m.EXPECT().ListAllHooks(gomock.Any()).Return(nil, nil)
			m.EXPECT().CreateHook(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(errors.New("test error"))

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := models.CreateWebhooksRequest{
				ConnectorID:    "test",
				WebhookBaseUrl: "http://localhost:8080",
			}

			m.EXPECT().ListAllHooks(gomock.Any()).Return(nil, nil)
			for range plg.webhookConfigs {
				m.EXPECT().CreateHook(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			}

			resp, err := plg.CreateWebhooks(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateWebhooksResponse{}))
		})
	})
})

var _ = Describe("Mangopay Plugin Translate Webhook", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create webhooks", func() {
		var (
			m                      *client.MockClient
			sampleTransferResponse client.TransferResponse
			samplePayoutResponse   client.PayoutResponse
			samplePayinResponse    client.PayinResponse
			sampleRefundResponse   client.Refund
			now                    time.Time
			date                   string
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()
			date = strconv.FormatInt(now.UTC().Unix(), 10)
			plg.initWebhookConfig()

			sampleTransferResponse = client.TransferResponse{
				ID:           "1",
				CreationDate: now.Unix(),
				AuthorID:     "u1",
				DebitedFunds: client.Funds{
					Currency: "EUR",
					Amount:   "100",
				},
				Fees: client.Funds{
					Currency: "EUR",
					Amount:   "0",
				},
				Status:           "SUCCEEDED",
				DebitedWalletID:  "acc1",
				CreditedWalletID: "acc2",
			}

			samplePayoutResponse = client.PayoutResponse{
				ID:           "1",
				CreationDate: now.Unix(),
				AuthorID:     "u1",
				DebitedFunds: client.Funds{
					Currency: "EUR",
					Amount:   "100",
				},
				Fees: client.Funds{
					Currency: "EUR",
					Amount:   "0",
				},
				Status:          "SUCCEEDED",
				BankAccountID:   "acc2",
				DebitedWalletID: "acc1",
			}

			samplePayinResponse = client.PayinResponse{
				ID:           "1",
				CreationDate: now.Unix(),
				AuthorId:     "u1",
				DebitedFunds: client.Funds{
					Currency: "EUR",
					Amount:   "100",
				},
				Fees: client.Funds{
					Currency: "EUR",
					Amount:   "0",
				},
				Status:           "SUCCEEDED",
				CreditedWalletID: "acc1",
			}

			sampleRefundResponse = client.Refund{
				ID:           "1",
				CreationDate: now.Unix(),
				AuthorId:     "u1",
				DebitedFunds: client.Funds{
					Currency: "EUR",
					Amount:   "100",
				},
				Fees: client.Funds{
					Currency: "EUR",
					Amount:   "0",
				},
				Status:                 "SUCEEDED",
				DebitedWalletId:        "acc2",
				CreditedWalletId:       "acc1",
				InitialTransactionID:   "123",
				InitialTransactionType: "PAYIN",
			}
		})

		It("should return an error - missing event type in query", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing EventType query parameter: invalid request"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - missing resource id in query", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType": {"TRANSFER_NORMAL_CREATED"},
					},
				},
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing RessourceId query parameter: invalid request"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - missing Date in query", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"TRANSFER_NORMAL_CREATED"},
						"RessourceId": {"1"},
					},
				},
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("missing Date query parameter: invalid request"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - invalid Date in query", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"TRANSFER_NORMAL_CREATED"},
						"RessourceId": {"1"},
						"Date":        {"test"},
					},
				},
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("invalid Date query parameter: strconv.ParseInt: parsing \"test\": invalid syntax: invalid request"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - invalid event type", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"TEST"},
						"RessourceId": {"1"},
						"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
					},
				},
			}

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("unsupported webhook event type: TEST: invalid request"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should return an error - get transfer error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"TRANSFER_NORMAL_CREATED"},
						"RessourceId": {"1"},
						"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
					},
				},
			}

			m.EXPECT().GetWalletTransfer(gomock.Any(), "1").Return(client.TransferResponse{}, errors.New("test error"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should be ok transfers", func(ctx SpecContext) {
			for _, eventType := range []string{
				"TRANSFER_NORMAL_CREATED",
				"TRANSFER_NORMAL_FAILED",
				"TRANSFER_NORMAL_SUCCEEDED",
			} {
				req := models.TranslateWebhookRequest{
					Name: "test",
					Webhook: models.PSPWebhook{
						QueryValues: map[string][]string{
							"EventType":   {eventType},
							"RessourceId": {"1"},
							"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
						},
					},
				}

				sa := sampleTransferResponse
				status := models.PAYMENT_STATUS_PENDING
				switch eventType {
				case "TRANSFER_NORMAL_FAILED":
					sa.Status = "FAILED"
					status = models.PAYMENT_STATUS_FAILED
				case "TRANSFER_NORMAL_SUCCEEDED":
					sa.Status = "SUCCEEDED"
					status = models.PAYMENT_STATUS_SUCCEEDED
				case "TRANSFER_NORMAL_CREATED":
					sa.Status = "CREATED"
					status = models.PAYMENT_STATUS_PENDING
				}
				raw, _ := json.Marshal(sa)

				m.EXPECT().GetWalletTransfer(gomock.Any(), "1").
					Return(sa, nil)

				resp, err := plg.TranslateWebhook(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(models.TranslateWebhookResponse{
					Responses: []models.WebhookResponse{
						{
							IdempotencyKey: fmt.Sprintf("1-%s-%s", eventType, date),
							Payment: &models.PSPPayment{
								Reference:                   "1",
								CreatedAt:                   time.Unix(sampleTransferResponse.CreationDate, 0),
								Type:                        models.PAYMENT_TYPE_TRANSFER,
								Amount:                      big.NewInt(100),
								Asset:                       "EUR/2",
								Scheme:                      models.PAYMENT_SCHEME_OTHER,
								Status:                      status,
								SourceAccountReference:      pointer.For("acc1"),
								DestinationAccountReference: pointer.For("acc2"),
								Raw:                         raw,
							},
						},
					},
				}))
			}
		})

		It("should return an error - get payout error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"PAYOUT_NORMAL_CREATED"},
						"RessourceId": {"1"},
						"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
					},
				},
			}

			m.EXPECT().GetPayout(gomock.Any(), "1").Return(nil, errors.New("test error"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should be ok payouts", func(ctx SpecContext) {
			for _, eventType := range []string{
				"PAYOUT_NORMAL_CREATED",
				"PAYOUT_NORMAL_FAILED",
				"PAYOUT_NORMAL_SUCCEEDED",
				"INSTANT_PAYOUT_FAILED",
				"INSTANT_PAYOUT_SUCCEEDED",
			} {
				req := models.TranslateWebhookRequest{
					Name: "test",
					Webhook: models.PSPWebhook{
						QueryValues: map[string][]string{
							"EventType":   {eventType},
							"RessourceId": {"1"},
							"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
						},
					},
				}

				sp := samplePayoutResponse
				status := models.PAYMENT_STATUS_PENDING
				switch eventType {
				case "PAYOUT_NORMAL_FAILED", "INSTANT_PAYOUT_FAILED":
					sp.Status = "FAILED"
					status = models.PAYMENT_STATUS_FAILED
				case "PAYOUT_NORMAL_SUCCEEDED", "INSTANT_PAYOUT_SUCCEEDED":
					sp.Status = "SUCCEEDED"
					status = models.PAYMENT_STATUS_SUCCEEDED
				case "PAYOUT_NORMAL_CREATED":
					sp.Status = "CREATED"
					status = models.PAYMENT_STATUS_PENDING
				}

				raw, _ := json.Marshal(sp)

				m.EXPECT().GetPayout(gomock.Any(), "1").
					Return(&sp, nil)

				resp, err := plg.TranslateWebhook(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(models.TranslateWebhookResponse{
					Responses: []models.WebhookResponse{
						{
							IdempotencyKey: fmt.Sprintf("1-%s-%s", eventType, date),
							Payment: &models.PSPPayment{
								Reference:                   "1",
								CreatedAt:                   time.Unix(samplePayoutResponse.CreationDate, 0),
								Type:                        models.PAYMENT_TYPE_PAYOUT,
								Amount:                      big.NewInt(100),
								Asset:                       "EUR/2",
								Scheme:                      models.PAYMENT_SCHEME_OTHER,
								Status:                      status,
								SourceAccountReference:      pointer.For("acc1"),
								DestinationAccountReference: pointer.For("acc2"),
								Raw:                         raw,
							},
						},
					},
				}))
			}
		})

		It("should return an error - get payin error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"PAYIN_NORMAL_CREATED"},
						"RessourceId": {"1"},
						"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
					},
				},
			}

			m.EXPECT().GetPayin(gomock.Any(), "1").Return(nil, errors.New("test error"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should be ok payins", func(ctx SpecContext) {
			for _, eventType := range []string{
				"PAYIN_NORMAL_CREATED",
				"PAYIN_NORMAL_FAILED",
				"PAYIN_NORMAL_SUCCEEDED",
			} {
				req := models.TranslateWebhookRequest{
					Name: "test",
					Webhook: models.PSPWebhook{
						QueryValues: map[string][]string{
							"EventType":   {eventType},
							"RessourceId": {"1"},
							"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
						},
					},
				}

				sp := samplePayinResponse
				status := models.PAYMENT_STATUS_PENDING
				switch eventType {
				case "PAYIN_NORMAL_FAILED":
					sp.Status = "FAILED"
					status = models.PAYMENT_STATUS_FAILED
				case "PAYIN_NORMAL_SUCCEEDED":
					sp.Status = "SUCCEEDED"
					status = models.PAYMENT_STATUS_SUCCEEDED
				case "PAYIN_NORMAL_CREATED":
					sp.Status = "CREATED"
					status = models.PAYMENT_STATUS_PENDING
				}

				raw, _ := json.Marshal(sp)

				m.EXPECT().GetPayin(gomock.Any(), "1").
					Return(&sp, nil)

				resp, err := plg.TranslateWebhook(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(models.TranslateWebhookResponse{
					Responses: []models.WebhookResponse{
						{
							IdempotencyKey: fmt.Sprintf("1-%s-%s", eventType, date),
							Payment: &models.PSPPayment{
								Reference:                   "1",
								CreatedAt:                   time.Unix(samplePayinResponse.CreationDate, 0),
								Type:                        models.PAYMENT_TYPE_PAYIN,
								Amount:                      big.NewInt(100),
								Asset:                       "EUR/2",
								Scheme:                      models.PAYMENT_SCHEME_OTHER,
								Status:                      status,
								DestinationAccountReference: pointer.For("acc1"),
								Raw:                         raw,
							},
						},
					},
				}))
			}
		})

		It("should return an error - get refund error", func(ctx SpecContext) {
			req := models.TranslateWebhookRequest{
				Name: "test",
				Webhook: models.PSPWebhook{
					QueryValues: map[string][]string{
						"EventType":   {"TRANSFER_REFUND_FAILED"},
						"RessourceId": {"1"},
						"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
					},
				},
			}

			m.EXPECT().GetRefund(gomock.Any(), "1").Return(nil, errors.New("test error"))

			resp, err := plg.TranslateWebhook(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.TranslateWebhookResponse{}))
		})

		It("should be ok refunds", func(ctx SpecContext) {
			for _, eventType := range []string{
				"TRANSFER_REFUND_FAILED",
				"TRANSFER_REFUND_SUCCEEDED",
				"PAYOUT_REFUND_FAILED",
				"PAYOUT_REFUND_SUCCEEDED",
				"PAYIN_REFUND_FAILED",
				"PAYIN_REFUND_SUCCEEDED",
			} {
				req := models.TranslateWebhookRequest{
					Name: "test",
					Webhook: models.PSPWebhook{
						QueryValues: map[string][]string{
							"EventType":   {eventType},
							"RessourceId": {"1"},
							"Date":        {strconv.FormatInt(now.UTC().Unix(), 10)},
						},
					},
				}

				sp := sampleRefundResponse
				status := models.PAYMENT_STATUS_PENDING
				pType := models.PAYMENT_TYPE_PAYIN
				switch eventType {
				case "TRANSFER_REFUND_FAILED":
					sp.Status = "FAILED"
					sp.InitialTransactionType = "TRANSFER"
					status = models.PAYMENT_STATUS_REFUNDED_FAILURE
					pType = models.PAYMENT_TYPE_TRANSFER
				case "TRANSFER_REFUND_SUCCEEDED":
					sp.Status = "SUCCEEDED"
					sp.InitialTransactionType = "TRANSFER"
					status = models.PAYMENT_STATUS_REFUNDED
					pType = models.PAYMENT_TYPE_TRANSFER
				case "PAYOUT_REFUND_FAILED":
					sp.Status = "FAILED"
					sp.InitialTransactionType = "PAYOUT"
					status = models.PAYMENT_STATUS_REFUNDED_FAILURE
					pType = models.PAYMENT_TYPE_PAYOUT
				case "PAYOUT_REFUND_SUCCEEDED":
					sp.Status = "SUCCEEDED"
					sp.InitialTransactionType = "PAYOUT"
					status = models.PAYMENT_STATUS_REFUNDED
					pType = models.PAYMENT_TYPE_PAYOUT
				case "PAYIN_REFUND_FAILED":
					sp.Status = "FAILED"
					sp.InitialTransactionType = "PAYIN"
					status = models.PAYMENT_STATUS_REFUNDED_FAILURE
					pType = models.PAYMENT_TYPE_PAYIN
				case "PAYIN_REFUND_SUCCEEDED":
					sp.Status = "SUCCEEDED"
					sp.InitialTransactionType = "PAYIN"
					status = models.PAYMENT_STATUS_REFUNDED
					pType = models.PAYMENT_TYPE_PAYIN
				}

				raw, _ := json.Marshal(sp)

				m.EXPECT().GetRefund(gomock.Any(), "1").
					Return(&sp, nil)

				resp, err := plg.TranslateWebhook(ctx, req)
				Expect(err).To(BeNil())
				Expect(resp).To(Equal(models.TranslateWebhookResponse{
					Responses: []models.WebhookResponse{
						{
							IdempotencyKey: fmt.Sprintf("1-%s-%s", eventType, date),
							Payment: &models.PSPPayment{
								ParentReference: "123",
								Reference:       "1",
								CreatedAt:       time.Unix(sampleRefundResponse.CreationDate, 0),
								Type:            pType,
								Amount:          big.NewInt(100),
								Asset:           "EUR/2",
								Scheme:          models.PAYMENT_SCHEME_OTHER,
								Status:          status,
								Raw:             raw,
							},
						},
					},
				}))
			}
		})
	})
})
