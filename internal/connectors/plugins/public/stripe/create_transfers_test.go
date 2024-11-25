package stripe

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/stripe/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stripe/stripe-go/v79"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Stripe Plugin Transfers Creation", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("create transfer", func() {
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

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("destination account is required: invalid request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - validation error - asset not supported", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Asset = "HHH/2"

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get currency and precision from asset: missing currencies: invalid request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - create transfer error", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().CreateTransfer(gomock.Any(), &client.CreateTransferRequest{
				IdempotencyKey: samplePSPPaymentInitiation.Reference,
				Amount:         100,
				Currency:       "EUR",
				Source:         pointer.For("acc1"),
				Destination:    "acc2",
				Description:    samplePSPPaymentInitiation.Description,
				Metadata:       samplePSPPaymentInitiation.Metadata,
			}).Return(nil, errors.New("test error"))

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("test error"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			trResponse := &stripe.Transfer{
				Amount: 100,
				BalanceTransaction: &stripe.BalanceTransaction{
					ID: "bt1",
				},
				Created:     now.Unix(),
				Currency:    "EUR",
				Description: samplePSPPaymentInitiation.Description,
				ID:          "t1",
				Metadata:    samplePSPPaymentInitiation.Metadata,
			}
			m.EXPECT().CreateTransfer(gomock.Any(), &client.CreateTransferRequest{
				IdempotencyKey: samplePSPPaymentInitiation.Reference,
				Amount:         100,
				Currency:       "EUR",
				Source:         pointer.For("acc1"),
				Destination:    "acc2",
				Description:    samplePSPPaymentInitiation.Description,
				Metadata:       samplePSPPaymentInitiation.Metadata,
			}).Return(trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateTransferResponse{
				Payment: &models.PSPPayment{
					Reference:                   "bt1",
					CreatedAt:                   time.Unix(trResponse.Created, 0),
					Type:                        models.PAYMENT_TYPE_TRANSFER,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("acc1"),
					DestinationAccountReference: pointer.For("acc2"),
					Metadata:                    samplePSPPaymentInitiation.Metadata,
					Raw:                         raw,
				},
			}))
		})
	})
})
