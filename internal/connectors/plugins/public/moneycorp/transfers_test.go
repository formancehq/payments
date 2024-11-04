package moneycorp

import (
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moneycorp Plugin Transfers Creation", func() {
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
				Reference:   "test1",
				CreatedAt:   now.UTC(),
				Description: "test1",
				SourceAccount: &models.PSPAccount{
					Reference:    "123",
					CreatedAt:    now.Add(-time.Duration(50) * time.Minute).UTC(),
					Name:         pointer.For("acc1"),
					DefaultAsset: pointer.For("EUR/2"),
				},
				DestinationAccount: &models.PSPAccount{
					Reference:    "321",
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
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("source account is required: invalid request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
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

			req.PaymentInitiation.Asset = "HUF/2"

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to get currency and precision from asset: missing currencies: invalid request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - initiate transfer error", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				IdempotencyKey:     samplePSPPaymentInitiation.Reference,
				SourceAccountID:    samplePSPPaymentInitiation.SourceAccount.Reference,
				ReceivingAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				TransferAmount:     "1.00",
				TransferCurrency:   "EUR",
				TransferReference:  samplePSPPaymentInitiation.Description,
				ClientReference:    samplePSPPaymentInitiation.Description,
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

			trResponse := client.TransferResponse{
				ID: "1",
				Attributes: client.TransferAttributes{
					SendingAccountID:   123,
					ReceivingAccountID: 321,
					CreatedAt:          now.Format("2006-01-02T15:04:05.999999999"),
					TransferReference:  samplePSPPaymentInitiation.Description,
					ClientReference:    samplePSPPaymentInitiation.Description,
					TransferAmount:     "1.00",
					TransferCurrency:   "EUR",
					TransferStatus:     "Cleared",
				},
			}
			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				IdempotencyKey:     samplePSPPaymentInitiation.Reference,
				SourceAccountID:    samplePSPPaymentInitiation.SourceAccount.Reference,
				ReceivingAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				TransferAmount:     "1.00",
				TransferCurrency:   "EUR",
				TransferReference:  samplePSPPaymentInitiation.Description,
				ClientReference:    samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateTransferResponse{
				Payment: models.PSPPayment{
					Reference:                   "1",
					CreatedAt:                   now,
					Type:                        models.PAYMENT_TYPE_TRANSFER,
					Amount:                      big.NewInt(100),
					Asset:                       "EUR/2",
					Scheme:                      models.PAYMENT_SCHEME_OTHER,
					Status:                      models.PAYMENT_STATUS_SUCCEEDED,
					SourceAccountReference:      pointer.For("123"),
					DestinationAccountReference: pointer.For("321"),
					Raw:                         raw,
				},
			}))
		})
	})
})
