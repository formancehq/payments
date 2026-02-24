package stripe

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/pkg/connectors/stripe/client"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v80"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin Transfers Reversal", func() {
	var (
		ctrl *gomock.Controller
		m    *client.MockClient
		plg  connector.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		m = client.NewMockClient(ctrl)
		plg = &Plugin{client: m}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("create transfer", func() {
		var (
			samplePSPPaymentInitiationReversal connector.PSPPaymentInitiationReversal
			now                                time.Time
		)
		BeforeEach(func() {
			now = time.Now().UTC()
			samplePSPPaymentInitiationReversal = connector.PSPPaymentInitiationReversal{
				Reference:   "test_reversal_1",
				CreatedAt:   now.UTC(),
				Description: "test_reversal_1",
				RelatedPaymentInitiation: connector.PSPPaymentInitiation{
					Reference:   uuid.New().String(),
					CreatedAt:   now.UTC(),
					Description: "test1",
					SourceAccount: &connector.PSPAccount{
						Reference:    "acc1",
						CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
						Name:         pointer.For("acc1"),
						DefaultAsset: pointer.For("EUR/2"),
						Metadata:     map[string]string{"userID": "u1"},
					},
					DestinationAccount: &connector.PSPAccount{
						Reference:    "acc2",
						CreatedAt:    now.Add(-time.Duration(49) * time.Minute).UTC(),
						Name:         pointer.For("acc2"),
						DefaultAsset: pointer.For("EUR/2"),
					},
					Amount:   big.NewInt(100),
					Asset:    "EUR/2",
					Metadata: map[string]string{"foo": "bar"},
				},
				Amount: big.NewInt(50),
				Asset:  "EUR/2",
				Metadata: map[string]string{
					"com.stripe.spec/transfer_id": "acc_test",
				},
			}
		})

		It("should return an error - validation error - missing metadata", func(ctx SpecContext) {
			c := samplePSPPaymentInitiationReversal
			delete(c.Metadata, "com.stripe.spec/transfer_id")
			req := connector.ReverseTransferRequest{
				PaymentInitiationReversal: samplePSPPaymentInitiationReversal,
			}
			resp, err := plg.ReverseTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("transfer id is required in metadata of transfer reversal request: invalid request"))
			Expect(resp).To(Equal(connector.ReverseTransferResponse{}))
		})
		It("should return an error - reverse transfer error", func(ctx SpecContext) {
			req := connector.ReverseTransferRequest{
				PaymentInitiationReversal: samplePSPPaymentInitiationReversal,
			}
			m.EXPECT().GetRootAccountID().Return("roooooot")
			m.EXPECT().ReverseTransfer(gomock.Any(), client.ReverseTransferRequest{
				IdempotencyKey:   samplePSPPaymentInitiationReversal.Reference,
				StripeTransferID: samplePSPPaymentInitiationReversal.Metadata["com.stripe.spec/transfer_id"],
				Account:          &samplePSPPaymentInitiationReversal.RelatedPaymentInitiation.SourceAccount.Reference,
				Amount:           samplePSPPaymentInitiationReversal.Amount.Int64(),
				Description:      samplePSPPaymentInitiationReversal.Description,
				Metadata:         samplePSPPaymentInitiationReversal.Metadata,
			}).Return(nil, errors.New("test error"))
			resp, err := plg.ReverseTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(connector.ReverseTransferResponse{}))
		})
		It("should be ok", func(ctx SpecContext) {
			req := connector.ReverseTransferRequest{
				PaymentInitiationReversal: samplePSPPaymentInitiationReversal,
			}
			trReversal := &stripe.TransferReversal{
				Amount: 100,
				BalanceTransaction: &stripe.BalanceTransaction{
					ID: "bt2",
				},
				Created:  now.Unix(),
				Currency: "eur",
				ID:       "tr1",
				Metadata: samplePSPPaymentInitiationReversal.Metadata,
				Transfer: &stripe.Transfer{
					Amount: 100,
					BalanceTransaction: &stripe.BalanceTransaction{
						ID: "bt1",
					},
					Created:     now.Unix(),
					Currency:    "EUR",
					Description: samplePSPPaymentInitiationReversal.RelatedPaymentInitiation.Description,
					ID:          "t1",
					Metadata:    samplePSPPaymentInitiationReversal.RelatedPaymentInitiation.Metadata,
				},
			}
			m.EXPECT().GetRootAccountID().Return("roooooot")
			m.EXPECT().ReverseTransfer(gomock.Any(), client.ReverseTransferRequest{
				IdempotencyKey:   samplePSPPaymentInitiationReversal.Reference,
				StripeTransferID: samplePSPPaymentInitiationReversal.Metadata["com.stripe.spec/transfer_id"],
				Account:          &samplePSPPaymentInitiationReversal.RelatedPaymentInitiation.SourceAccount.Reference,
				Amount:           samplePSPPaymentInitiationReversal.Amount.Int64(),
				Description:      samplePSPPaymentInitiationReversal.Description,
				Metadata:         samplePSPPaymentInitiationReversal.Metadata,
			}).Return(trReversal, nil)
			raw, err := json.Marshal(&trReversal)
			Expect(err).To(BeNil())
			resp, err := plg.ReverseTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(connector.ReverseTransferResponse{
				Payment: connector.PSPPayment{
					ParentReference:             "bt1",
					Reference:                   "bt2",
					CreatedAt:                   time.Unix(trReversal.Created, 0),
					Type:                        connector.PAYMENT_TYPE_TRANSFER,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      connector.PAYMENT_SCHEME_OTHER,
					Status:                      connector.PAYMENT_STATUS_REFUNDED,
					SourceAccountReference:      pointer.For("acc1"),
					DestinationAccountReference: pointer.For("acc2"),
					Metadata:                    samplePSPPaymentInitiationReversal.Metadata,
					Raw:                         raw,
				},
			}))
		})
	})
})
