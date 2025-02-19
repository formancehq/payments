package increase

import (
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

var _ = Describe("Increase Plugin Transfers Creation", func() {
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
			trResponse                 client.TransferResponse
			payment                    *models.PSPPayment
			now                        time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg.client = m
			now, _ = time.Parse(time.RFC3339, time.Now().UTC().Format(time.RFC3339))

			samplePSPPaymentInitiation = models.PSPPaymentInitiation{
				Reference:   "test1",
				CreatedAt:   now.UTC(),
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
					"foo": "bar",
				},
			}

			trResponse = client.TransferResponse{
				ID:                   "1",
				Status:               "complete",
				CreatedAt:            now.Format(time.RFC3339),
				Description:          samplePSPPaymentInitiation.Description,
				Currency:             "USD",
				DestinationAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				AccountID:            samplePSPPaymentInitiation.SourceAccount.Reference,
				Amount:               "1.00",
			}

			payment = &models.PSPPayment{
				Reference:                   "1",
				CreatedAt:                   now,
				Type:                        models.PAYMENT_TYPE_TRANSFER,
				Amount:                      big.NewInt(100),
				Asset:                       "USD/2",
				Scheme:                      models.PAYMENT_SCHEME_OTHER,
				Status:                      models.PAYMENT_STATUS_SUCCEEDED,
				SourceAccountReference:      pointer.For("acc1"),
				DestinationAccountReference: pointer.For("acc2"),
			}
		})

		It("should return an error - validation error - amount", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Amount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("amount is required: invalid request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - validation error - description", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Description = ""

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("description is required: invalid request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
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
				AccountID:            samplePSPPaymentInitiation.SourceAccount.Reference,
				DestinationAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:               "1.00",
				Description:          samplePSPPaymentInitiation.Description,
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

			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				AccountID:            samplePSPPaymentInitiation.SourceAccount.Reference,
				DestinationAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:               "1.00",
				Description:          samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			payment.Raw = raw
			payment.Status = models.PAYMENT_STATUS_SUCCEEDED
			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateTransferResponse{
				Payment: payment,
			}))
		})

		It("should be ok - submitted status", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				AccountID:            samplePSPPaymentInitiation.SourceAccount.Reference,
				DestinationAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:               "1.00",
				Description:          samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

			trResponse.Status = "submitted"
			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			payment.Raw = raw
			payment.Status = models.PAYMENT_STATUS_PENDING
			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateTransferResponse{
				Payment: payment,
			}))
		})

		It("should be ok - canceled status", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			m.EXPECT().InitiateTransfer(gomock.Any(), &client.TransferRequest{
				AccountID:            samplePSPPaymentInitiation.SourceAccount.Reference,
				DestinationAccountID: samplePSPPaymentInitiation.DestinationAccount.Reference,
				Amount:               "1.00",
				Description:          samplePSPPaymentInitiation.Description,
			}).Return(&trResponse, nil)

			trResponse.Status = "canceled"
			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			payment.Raw = raw
			payment.Status = models.PAYMENT_STATUS_CANCELLED
			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateTransferResponse{
				Payment: payment,
			}))
		})
	})
})
