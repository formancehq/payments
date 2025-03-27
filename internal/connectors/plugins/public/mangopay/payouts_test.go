package mangopay

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Mangopay Plugin Payouts Creation", func() {
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

		It("should return an error - validation error - reference", func(ctx SpecContext) {
			sa := samplePSPPaymentInitiation
			sa.Reference = "test"
			req := models.CreatePayoutRequest{
				PaymentInitiation: sa,
			}

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("reference test is required to be an uuid in payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - source account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account is required in payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account is required in payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - missing user ID in source account", func(ctx SpecContext) {
			sa := samplePSPPaymentInitiation
			sa.SourceAccount.Metadata = map[string]string{}
			req := models.CreatePayoutRequest{
				PaymentInitiation: sa,
			}

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account metadata with user id is required in payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - description error", func(ctx SpecContext) {
			sa := samplePSPPaymentInitiation
			sa.Description = "test1test349[;'/';']"
			req := models.CreatePayoutRequest{
				PaymentInitiation: sa,
			}

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("description must be alphanumeric and less than 12 characters in payout request: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - validation error - asset not supported", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Asset = "HUF/2"

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get currency and precision from asset: HUF: missing currencies: invalid request"))
			Expect(resp).To(Equal(models.CreatePayoutResponse{}))
		})

		It("should return an error - initiate payout error", func(ctx SpecContext) {
			req := models.CreatePayoutRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().InitiatePayout(gomock.Any(), &client.PayoutRequest{
				Reference: samplePSPPaymentInitiation.Reference,
				AuthorID:  "u1",
				DebitedFunds: client.Funds{
					Currency: "EUR",
					Amount:   "100",
				},
				Fees: client.Funds{
					Currency: "EUR",
					Amount:   "0",
				},
				DebitedWalletID: samplePSPPaymentInitiation.SourceAccount.Reference,
				BankAccountID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				BankWireRef:     "test1",
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

			trResponse := client.PayoutResponse{
				ID:           "123",
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
				BankAccountID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				DebitedWalletID: samplePSPPaymentInitiation.SourceAccount.Reference,
			}
			m.EXPECT().InitiatePayout(gomock.Any(), &client.PayoutRequest{
				Reference: samplePSPPaymentInitiation.Reference,
				AuthorID:  "u1",
				DebitedFunds: client.Funds{
					Currency: "EUR",
					Amount:   "100",
				},
				Fees: client.Funds{
					Currency: "EUR",
					Amount:   "0",
				},
				DebitedWalletID: samplePSPPaymentInitiation.SourceAccount.Reference,
				BankAccountID:   samplePSPPaymentInitiation.DestinationAccount.Reference,
				BankWireRef:     "test1",
			}).Return(&trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreatePayout(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreatePayoutResponse{
				Payment: &models.PSPPayment{
					Reference:                   "123",
					CreatedAt:                   time.Unix(trResponse.CreationDate, 0),
					Type:                        models.PAYMENT_TYPE_PAYOUT,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("acc1"),
					DestinationAccountReference: pointer.For("acc2"),
					Raw:                         raw,
				},
			}))
		})

	})
})
