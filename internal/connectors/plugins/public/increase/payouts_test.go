package increase

import (
	// "encoding/json"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
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
			m                          *client.MockClient
			samplePSPPaymentInitiation models.PSPPaymentInitiation
			now                        time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now, _ = time.Parse("2006-01-02T15:04:05.999-0700", time.Now().UTC().Format("2006-01-02T15:04:05.999-0700"))

			samplePSPPaymentInitiation = models.PSPPaymentInitiation{
				Reference:   "test1",
				CreatedAt:   now,
				Description: "test1",
				SourceAccount: &models.PSPAccount{
					Reference:    "acc1",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("USD/2"),
					Metadata: map[string]string{
						client.IncreaseSourceAccountNumberIdMetadataKey: "123456789",
					},
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
					client.IncreaseFufillmentMethodMetadataKey: "third_party",
					client.IncreaseCheckNumberMetadataKey:      "123456789",
					client.IncreasePayoutMethodMetadataKey:     "ach",
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
			Expect(err).To(MatchError("source account is required: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account is required: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - payment method is required", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("payoutMethod is a required metadata: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - payout method must be one of: ach, wire, check, rtp", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "test"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("payoutMethod must be one of: ach, wire, check, rtp: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate ach payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "ach"

			m.EXPECT().InitiateACHTransferPayout(gomock.Any(), &client.ACHPayoutRequest{
				AccountID:           samplePSPPaymentInitiation.SourceAccount.Reference,
				ExternalAccountID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:              "1.00",
				IndividualName:      *samplePSPPaymentInitiation.DestinationAccount.Name,
				StatementDescriptor: samplePSPPaymentInitiation.Description,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate wire payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "wire"

			m.EXPECT().InitiateWireTransferPayout(gomock.Any(), &client.WireTransferPayoutRequest{
				AccountID:          samplePSPPaymentInitiation.SourceAccount.Reference,
				ExternalAccountID:  samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:             "1.00",
				BeneficiaryName:    *samplePSPPaymentInitiation.DestinationAccount.Name,
				MessageToRecipient: samplePSPPaymentInitiation.Description,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate check payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "check"

			m.EXPECT().InitiateCheckTransferPayout(gomock.Any(), &client.CheckPayoutRequest{
				AccountID:             samplePSPPaymentInitiation.SourceAccount.Reference,
				SourceAccountNumberID: samplePSPPaymentInitiation.SourceAccount.Metadata[client.IncreaseSourceAccountNumberIdMetadataKey], // Changed from DestinationAccount.Reference
				FulfillmentMethod:     thirdPartyFufillmentMethod,
				Amount:                "1.00",
				ThirdParty: struct {
					CheckNumber string `json:"check_number"`
				}{
					CheckNumber: samplePSPPaymentInitiation.Metadata[client.IncreaseCheckNumberMetadataKey],
				},
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate rtp payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "rtp"

			m.EXPECT().InitiateRTPTransferPayout(gomock.Any(), &client.RTPPayoutRequest{
				SourceAccountNumberID: samplePSPPaymentInitiation.SourceAccount.Metadata[client.IncreaseSourceAccountNumberIdMetadataKey],
				ExternalAccountID:     samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:                "1.00",
				CreditorName:          *samplePSPPaymentInitiation.DestinationAccount.Name,
				RemittanceInformation: samplePSPPaymentInitiation.Description,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should be ok - ach", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format("2006-01-02T15:04:05.999-0700"),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            "1.00",
			}
			m.EXPECT().InitiateACHTransferPayout(gomock.Any(), &client.ACHPayoutRequest{
				AccountID:           samplePSPPaymentInitiation.SourceAccount.Reference,
				ExternalAccountID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:              "1.00",
				IndividualName:      *samplePSPPaymentInitiation.DestinationAccount.Name,
				StatementDescriptor: samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

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
					Metadata:                    map[string]string{"routingNumber": "", "accountNumber": "", "recipientName": "", "checkNumber": ""},
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
				CreatedAt:         now.Format("2006-01-02T15:04:05.999-0700"),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            "1.00",
			}
		
			m.EXPECT().InitiateWireTransferPayout(gomock.Any(), &client.WireTransferPayoutRequest{
				AccountID:          samplePSPPaymentInitiation.SourceAccount.Reference,
				ExternalAccountID:  samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:            "1.00",
				BeneficiaryName:   *samplePSPPaymentInitiation.DestinationAccount.Name,
				MessageToRecipient: samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)
		
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
					Metadata:                    map[string]string{"routingNumber": "", "accountNumber": "", "recipientName": "", "checkNumber": ""},
				},
			}))
		})

		It("should be ok - check", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}
			req.PaymentInitiation.Metadata[client.IncreasePayoutMethodMetadataKey] = "check"
		
			trResponse := client.PayoutResponse{
				ID:                "1",
				Status:            "complete",
				CreatedAt:         now.Format("2006-01-02T15:04:05.999-0700"),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            "1.00",
			}
		
			m.EXPECT().InitiateCheckTransferPayout(gomock.Any(), &client.CheckPayoutRequest{
				AccountID:             samplePSPPaymentInitiation.SourceAccount.Reference,
				SourceAccountNumberID: samplePSPPaymentInitiation.SourceAccount.Metadata[client.IncreaseSourceAccountNumberIdMetadataKey],
				FulfillmentMethod:     thirdPartyFufillmentMethod,
				Amount:                "1.00",
				ThirdParty: struct {
					CheckNumber string `json:"check_number"`
				}{
					CheckNumber: samplePSPPaymentInitiation.Metadata[client.IncreaseCheckNumberMetadataKey],
				},
			}).Return(&trResponse, nil)
		
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
					Metadata:                    map[string]string{"routingNumber": "", "accountNumber": "", "recipientName": "", "checkNumber": ""},
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
				CreatedAt:         now.Format("2006-01-02T15:04:05.999-0700"),
				AccountID:         "234R5432",
				ExternalAccountId: "acc2",
				Currency:          "USD",
				Amount:            "1.00",
			}
		
			m.EXPECT().InitiateRTPTransferPayout(gomock.Any(), &client.RTPPayoutRequest{
				SourceAccountNumberID: samplePSPPaymentInitiation.SourceAccount.Metadata[client.IncreaseSourceAccountNumberIdMetadataKey],
				ExternalAccountID:     samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:                "1.00",
				CreditorName:          *samplePSPPaymentInitiation.DestinationAccount.Name,
				RemittanceInformation: samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)
		
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
					Metadata:                    map[string]string{"routingNumber": "", "accountNumber": "", "recipientName": "", "checkNumber": ""},
				},
			}))
		})		
	})
})
