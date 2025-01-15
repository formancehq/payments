package atlar

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/atlar/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
	"github.com/get-momo/atlar-v1-go-client/client/credit_transfers"
	"github.com/get-momo/atlar-v1-go-client/client/third_parties"
	"github.com/get-momo/atlar-v1-go-client/client/transactions"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Atlar Plugin Payouts Creation", func() {
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
			now = time.Now().UTC()

			samplePSPPaymentInitiation = models.PSPPaymentInitiation{
				Reference:   uuid.New().String(),
				CreatedAt:   now.UTC(),
				Description: "test1",
				SourceAccount: &models.PSPAccount{
					Reference:    "acc1",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("EUR/2"),
					Metadata: map[string]string{
						"userID": "u1",
					},
				},
				DestinationAccount: &models.PSPAccount{
					Reference:    "acc2",
					CreatedAt:    now.Add(-time.Duration(49) * time.Minute).UTC(),
					Name:         pointer.For("acc2"),
					DefaultAsset: pointer.For("EUR/2"),
				},
				Amount: big.NewInt(100),
				Asset:  "EUR/2",
				Metadata: map[string]string{
					"foo": "bar",
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

		It("should return an error - validation error - asset not supported", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Asset = "HUF/2"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get currency and precision from asset: missing currencies: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().PostV1CreditTransfers(gomock.Any(), &atlar_models.CreatePaymentRequest{
				Amount: &atlar_models.AmountInput{
					Currency:    pointer.For("EUR"),
					StringValue: "1.00",
					Value:       100,
				},
				Date:                         pointer.For(samplePSPPaymentInitiation.CreatedAt.Format(time.DateOnly)),
				DestinationExternalAccountID: &samplePSPPaymentInitiation.DestinationAccount.Reference,
				ExternalID:                   samplePSPPaymentInitiation.Reference,
				PaymentSchemeType:            pointer.For("SCT"),
				RemittanceInformation: &atlar_models.RemittanceInformation{
					Type:  pointer.For("UNSTRUCTURED"),
					Value: &samplePSPPaymentInitiation.Description,
				},
				SourceAccountID: &samplePSPPaymentInitiation.SourceAccount.Reference,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().PostV1CreditTransfers(gomock.Any(), &atlar_models.CreatePaymentRequest{
				Amount: &atlar_models.AmountInput{
					Currency:    pointer.For("EUR"),
					StringValue: "1.00",
					Value:       100,
				},
				Date:                         pointer.For(samplePSPPaymentInitiation.CreatedAt.Format(time.DateOnly)),
				DestinationExternalAccountID: &samplePSPPaymentInitiation.DestinationAccount.Reference,
				ExternalID:                   samplePSPPaymentInitiation.Reference,
				PaymentSchemeType:            pointer.For("SCT"),
				RemittanceInformation: &atlar_models.RemittanceInformation{
					Type:  pointer.For("UNSTRUCTURED"),
					Value: &samplePSPPaymentInitiation.Description,
				},
				SourceAccountID: &samplePSPPaymentInitiation.SourceAccount.Reference,
			}).Return(nil, nil)

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				PollingPayoutID: &samplePSPPaymentInitiation.Reference,
			}))
		})
	})

	Context("poll payout status", func() {
		var (
			m                      *client.MockClient
			payoutID               string
			creditTransferResponse credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK
			now                    time.Time
			sampleAccount          atlar_models.Account
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now = time.Now().UTC()
			payoutID = "test"

			creditTransferResponse = credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK{
				Payload: &atlar_models.Payment{
					ID: payoutID,
					Reconciliation: &atlar_models.ReconciliationDetails{
						BookedTransactionID: "test-transaction",
					},
					Status: "CREATED",
				},
			}

			sampleAccount = atlar_models.Account{
				Bank: &atlar_models.BankSlim{
					Bic: "test",
				},
				Bic:          "test",
				ThirdPartyID: "test",
			}
		})

		It("should return an error - get credit transfer error", func(ctx SpecContext) {
			m.EXPECT().GetV1CreditTransfersGetByExternalIDExternalID(gomock.Any(), "test").Return(nil, errors.New("test error"))

			resp, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{
				PayoutID: payoutID,
			})
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.PollPayoutStatusResponse{}))
		})

		It("should return nil - payment still pending", func(ctx SpecContext) {
			m.EXPECT().GetV1CreditTransfersGetByExternalIDExternalID(gomock.Any(), "test").Return(&creditTransferResponse, nil)

			resp, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{
				PayoutID: payoutID,
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.PollPayoutStatusResponse{
				Payment: nil,
				Error:   nil,
			}))
		})

		It("should return an error - payment failed", func(ctx SpecContext) {
			c := creditTransferResponse
			c.Payload.Status = "REJECTED"
			m.EXPECT().GetV1CreditTransfersGetByExternalIDExternalID(gomock.Any(), "test").Return(&c, nil)

			resp, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{
				PayoutID: payoutID,
			})
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.PollPayoutStatusResponse{
				Payment: nil,
				Error:   pointer.For("payment failed: REJECTED"),
			}))
		})

		It("should return an error - unknown status", func(ctx SpecContext) {
			c := creditTransferResponse
			c.Payload.Status = "UNKNOWN"
			m.EXPECT().GetV1CreditTransfersGetByExternalIDExternalID(gomock.Any(), "test").Return(&c, nil)

			resp, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{
				PayoutID: payoutID,
			})
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("unknown status \"UNKNOWN\" encountered while fetching payment initiation status of payment \"test\""))
			Expect(resp).To(Equal(models.PollPayoutStatusResponse{}))
		})

		It("should should be ok", func(ctx SpecContext) {
			c := creditTransferResponse
			c.Payload.Status = "RECONCILED"
			m.EXPECT().GetV1CreditTransfersGetByExternalIDExternalID(gomock.Any(), "test").Return(&c, nil)

			m.EXPECT().GetV1TransactionsID(gomock.Any(), "test-transaction").Return(&transactions.GetV1TransactionsIDOK{
				Payload: &atlar_models.Transaction{
					ID: "test-transaction",
					Amount: &atlar_models.Amount{
						Currency:    pointer.For("EUR"),
						StringValue: pointer.For("100"),
						Value:       pointer.For(int64(100)),
					},
					Account: &atlar_models.AccountTrx{
						ID: pointer.For("test"),
					},
					Characteristics: &atlar_models.TransactionCharacteristics{
						BankTransactionCode: &atlar_models.BankTransactionCode{
							Description: "test",
							Domain:      "test",
							Family:      "test",
							Subfamily:   "test",
						},
					},
					Created:     now.UTC().Format(time.RFC3339Nano),
					Description: "test",
					Reconciliation: &atlar_models.ReconciliationDetails{
						BookedTransactionID:   "test",
						ExpectedTransactionID: "test",
						Status:                "test",
						TransactableID:        "test",
						TransactableType:      "test",
					},
					RemittanceInformation: &atlar_models.RemittanceInformation{
						Type:  pointer.For("test"),
						Value: pointer.For("test"),
					},
				},
			}, nil)

			m.EXPECT().GetV1AccountsID(gomock.Any(), "test").Return(
				&accounts.GetV1AccountsIDOK{
					Payload: &sampleAccount,
				},
				nil,
			)

			m.EXPECT().GetV1BetaThirdPartiesID(gomock.Any(), sampleAccount.ThirdPartyID).Return(
				&third_parties.GetV1betaThirdPartiesIDOK{
					Payload: &atlar_models.ThirdParty{
						ID:   "test",
						Name: "test",
					},
				},
				nil,
			)

			resp, err := plg.PollPayoutStatus(ctx, models.PollPayoutStatusRequest{
				PayoutID: payoutID,
			})
			Expect(err).To(BeNil())
			Expect(resp.Payment).ToNot(BeNil())
		})
	})
})
