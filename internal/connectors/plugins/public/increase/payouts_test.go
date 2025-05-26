package increase

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Increase Plugin Payouts Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create payout", func() {
		var (
			mockHTTPClient             *client.MockHTTPClient
			samplePSPPaymentInitiation models.PSPPaymentInitiation
			now                        time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com", "we5432345")
			plg.client.SetHttpClient(mockHTTPClient)
			now, _ = time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))

			samplePSPPaymentInitiation = models.PSPPaymentInitiation{
				Reference:   "test1",
				CreatedAt:   now,
				Description: "test1",
				SourceAccount: &models.PSPAccount{
					Reference:    "acc1",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("USD/2"),
				},
				DestinationAccount: &models.PSPAccount{
					Reference:    "acc2",
					CreatedAt:    now.Add(-time.Duration(49) * time.Minute).UTC(),
					Name:         pointer.For("acc2"),
					DefaultAsset: pointer.For("USD/2"),
				},
				Amount: big.NewInt(100),
				Asset:  "USD/2",
				Metadata: map[string]string{
					client.IncreaseFulfillmentMethodMetadataKey:     "third_party",
					client.IncreaseCheckNumberMetadataKey:           "123456789",
					client.IncreasePayoutMethodMetadataKey:          "ach",
					client.IncreaseSourceAccountNumberIdMetadataKey: "123456789",
				},
			}
		})

		It("should return an error - validation error - source account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field sourceAccount: missing required field in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field destinationAccount: missing required field in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - payment method is required", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/payoutMethod: missing required metadata in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - description is required", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Description = ""

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field description: missing required field in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - amount is required", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Amount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field amount: missing required field in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - fulfillmentMethod is required", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = increaseCheckPaymentMethod
			req.PaymentInitiation.Metadata[client.IncreaseFulfillmentMethodMetadataKey] = ""

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/fulfillmentMethod: missing required metadata in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - sourceAccountNumberID is required", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreaseSourceAccountNumberIdMetadataKey] = ""
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = increaseRTPPaymentMethod

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/sourceAccountNumberID: missing required metadata in request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - payout method must be one of: ach, wire, check, rtp", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "test"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field com.increase.spec/payoutMethod: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate ach payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "ach"

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create ach payout: test error : : status code: 0"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate wire payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "wire"

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create wire transfer payout: test error : : status code: 0"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate check payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "check"

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create check transfer payout: test error : : status code: 0"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate rtp payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "rtp"

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to create real time payments transfer payout: test error : : status code: 0"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should be ok - ach", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "ach"

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            100,
				AccountNumber:     "123456789",
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.ACHPayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				Currency:          "USD",
				Amount:            100,
				ExternalAccountID: "acc2",
				AccountNumber:     "123456789",
			})

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "USD/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("234R5432"),
					DestinationAccountReference: pointer.For("acc2"),
					Raw:                         raw,
					Metadata: map[string]string{
						client.IncreaseRoutingNumberMetadataKey: "",
						client.IncreaseAccountNumberMetadataKey: "123456789",
						client.IncreaseRecipientNameMetadataKey: "",
						client.IncreaseCheckNumberMetadataKey:   "",
					},
				},
			}))
		})

		It("should be ok - wire", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "wire"

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            100,
				AccountNumber:     "123456789",
				RoutingNumber:     "123456789",
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.WireTransferPayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				Currency:          "USD",
				Amount:            100,
				ExternalAccountID: "acc2",
				AccountNumber:     "123456789",
				RoutingNumber:     "123456789",
			})

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "USD/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("234R5432"),
					DestinationAccountReference: pointer.For("acc2"),
					Raw:                         raw,
					Metadata: map[string]string{
						client.IncreaseRoutingNumberMetadataKey: "123456789",
						client.IncreaseAccountNumberMetadataKey: "123456789",
						client.IncreaseRecipientNameMetadataKey: "",
						client.IncreaseCheckNumberMetadataKey:   "",
					},
				},
			}))
		})

		It("should be ok - check with thirdparty fulfillment", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "check"

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				Currency:          "USD",
				Amount:            100,
				RecipientName:     "Unknown",
				ExternalAccountId: "Unknown",
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CheckPayoutResponse{
				ID:        "1",
				AccountID: "234R5432",
				Status:    "complete",
				CreatedAt: now.Format(time.RFC3339),
				Amount:    100,
				Currency:  "USD",
			})

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "USD/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("234R5432"),
					DestinationAccountReference: pointer.For("Unknown"),
					Raw:                         raw,
					Metadata: map[string]string{
						client.IncreaseRoutingNumberMetadataKey: "",
						client.IncreaseAccountNumberMetadataKey: "",
						client.IncreaseRecipientNameMetadataKey: "Unknown",
						client.IncreaseCheckNumberMetadataKey:   "",
					},
				},
			}))
		})

		It("should be ok - check with physical fufillment", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = increaseCheckPaymentMethod
			req.PaymentInitiation.Metadata[client.IncreaseFulfillmentMethodMetadataKey] = physicalCheckFulfillmentMethod

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				Currency:          "USD",
				Amount:            100,
				RecipientName:     "Unknown",
				ExternalAccountId: "Unknown",
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.CheckPayoutResponse{
				ID:        "1",
				AccountID: "234R5432",
				Status:    "complete",
				CreatedAt: now.Format(time.RFC3339),
				Amount:    100,
				Currency:  "USD",
			})

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "USD/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("234R5432"),
					DestinationAccountReference: pointer.For("Unknown"),
					Raw:                         raw,
					Metadata: map[string]string{
						client.IncreaseRoutingNumberMetadataKey: "",
						client.IncreaseAccountNumberMetadataKey: "",
						client.IncreaseRecipientNameMetadataKey: "Unknown",
						client.IncreaseCheckNumberMetadataKey:   "",
					},
				},
			}))
		})

		It("should be ok - rtp", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "rtp"

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            100,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, client.RTPPayoutResponse{
				ID:                "1",
				AccountID:         "234R5432",
				Status:            "complete",
				CreatedAt:         now.Format(time.RFC3339),
				Amount:            100,
				Currency:          "USD",
				ExternalAccountID: "acc2",
			})

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "USD/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("234R5432"),
					DestinationAccountReference: pointer.For("acc2"),
					Raw:                         raw,
					Metadata: map[string]string{
						client.IncreaseRoutingNumberMetadataKey: "",
						client.IncreaseAccountNumberMetadataKey: "",
						client.IncreaseRecipientNameMetadataKey: "",
						client.IncreaseCheckNumberMetadataKey:   "",
					},
				},
			}))
		})
	})
})
