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
	"github.com/golang/mock/gomock"
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
			mockHTTPClient             *client.MockHTTPClient
			samplePSPPaymentInitiation models.PSPPaymentInitiation
			trResponse                 client.TransferResponse
			payment                    *models.PSPPayment
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
				Amount:               100,
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
				Metadata: map[string]string{
					client.IncreaseDescriptionMetadataKey:              "test1",
					client.IncreaseTransactionIDMetadataKey:            "",
					client.IncreaseDestinationTransactionIDMetadataKey: "",
				},
			}
		})

		It("should return an error - validation error - amount", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Amount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field amount: missing required field in request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - validation error - description", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.Description = ""

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field description: missing required field in request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - validation error - source account", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.SourceAccount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field sourceAccount: missing required field in request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - validation error - destination account", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			req.PaymentInitiation.DestinationAccount = nil

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("validation error occurred for field destinationAccount: missing required field in request"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should return an error - initiate transfer error", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				500,
				errors.New("test error"),
			)

			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err).To(MatchError("failed to initiate transfer: test error : : status code: 0"))
			Expect(resp).To(Equal(models.CreateTransferResponse{}))
		})

		It("should be ok", func(ctx SpecContext) {
			req := models.CreateTransferRequest{
				PaymentInitiation: samplePSPPaymentInitiation,
			}

			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, trResponse)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			payment.Raw = raw
			payment.Status = models.PAYMENT_STATUS_SUCCEEDED
			payment.ParentReference = "1"
			payment.Reference = "1"
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

			trResponse.Status = "submitted"
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, trResponse)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			payment.Raw = raw
			payment.Status = models.PAYMENT_STATUS_SUCCEEDED
			payment.ParentReference = "1"
			payment.Reference = "1"
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

			trResponse.Status = "canceled"
			mockHTTPClient.EXPECT().Do(
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(
				200,
				nil,
			).SetArg(2, trResponse)

			raw, err := json.Marshal(&trResponse)
			Expect(err).To(BeNil())

			payment.Raw = raw
			payment.Status = models.PAYMENT_STATUS_CANCELLED
			payment.ParentReference = "1"
			payment.Reference = "1"
			resp, err := plg.CreateTransfer(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(models.CreateTransferResponse{
				Payment: payment,
			}))
		})
	})
})
