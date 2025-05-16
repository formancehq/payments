package qonto

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/formancehq/go-libs/v3/logging"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Qonto *Plugin Transfer", func() {
	Context("create transfer", func() {
		var (
			plg               *Plugin
			m                 *client.MockClient
			paymentInitiation models.PSPPaymentInitiation
			transferResponse  client.TransferResponse
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			m = client.NewMockClient(ctrl)
			plg = &Plugin{
				client: m,
				logger: logging.NewDefaultLogger(GinkgoWriter, true, false, false),
			}
			paymentInitiation = models.PSPPaymentInitiation{
				Reference: "new-transfer",
				SourceAccount: &models.PSPAccount{
					Reference: "source-account",
					Metadata: map[string]string{
						"bank_account_iban": "source-iban",
					},
				},
				DestinationAccount: &models.PSPAccount{
					Reference: "dest-account",
					Metadata: map[string]string{
						"bank_account_iban": "desc-iban",
					},
				},
				Amount: big.NewInt(100),
				Asset:  "EUR/2",
			}
			transferResponse = client.TransferResponse{
				Id:          "123456789",
				Slug:        "slug",
				Status:      "processing",
				Amount:      "1",
				AmountCents: "100",
				Currency:    "EUR",
				Reference:   "external-reference",
				CreatedDate: "2021-01-01T00:00:00.001Z",
			}
		})

		It("creates a transfer", func(ctx SpecContext) {
			// Given a valid request & client's response
			m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).
				Times(1).
				Return(&transferResponse, nil)

			// When
			resp, err := plg.createTransfer(ctx, paymentInitiation)

			// Then
			Expect(err).To(BeNil())

			raw, _ := json.Marshal(transferResponse)
			createdAt, _ := time.ParseInLocation(client.QONTO_TIMEFORMAT, transferResponse.CreatedDate, time.UTC)
			expectedPSPPayment := models.PSPPayment{
				Reference:                   transferResponse.Id,
				Type:                        models.PAYMENT_TYPE_TRANSFER,
				CreatedAt:                   createdAt,
				Amount:                      paymentInitiation.Amount,
				Asset:                       paymentInitiation.Asset,
				Scheme:                      models.PAYMENT_SCHEME_SEPA,
				Status:                      models.PAYMENT_STATUS_PENDING,
				SourceAccountReference:      &paymentInitiation.SourceAccount.Reference,
				DestinationAccountReference: &paymentInitiation.DestinationAccount.Reference,
				Metadata: map[string]string{
					"external_reference": transferResponse.Reference,
				},
				Raw: raw,
			}
			Expect(resp).To(Equal(&expectedPSPPayment))
		})

		It("defaults to EUR if the currency is not part of the response", func(ctx SpecContext) {
			// Given a valid request but client's response doesn't have currency set
			transferResponse.Currency = ""
			m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).
				Times(1).
				Return(&transferResponse, nil)

			// When
			resp, err := plg.createTransfer(ctx, paymentInitiation)

			// Then
			Expect(err).To(BeNil())

			raw, _ := json.Marshal(transferResponse)
			createdAt, _ := time.ParseInLocation(client.QONTO_TIMEFORMAT, transferResponse.CreatedDate, time.UTC)
			expectedPSPPayment := models.PSPPayment{
				Reference:                   transferResponse.Id,
				Type:                        models.PAYMENT_TYPE_TRANSFER,
				CreatedAt:                   createdAt,
				Amount:                      paymentInitiation.Amount,
				Asset:                       paymentInitiation.Asset,
				Scheme:                      models.PAYMENT_SCHEME_SEPA,
				Status:                      models.PAYMENT_STATUS_PENDING,
				SourceAccountReference:      &paymentInitiation.SourceAccount.Reference,
				DestinationAccountReference: &paymentInitiation.DestinationAccount.Reference,
				Metadata: map[string]string{
					"external_reference": transferResponse.Reference,
				},
				Raw: raw,
			}
			Expect(resp).To(Equal(&expectedPSPPayment))
		})

		Describe("Invalid requests cases", func() {
			It("Missing amount", func(ctx SpecContext) {
				// given
				paymentInitiation.Amount = nil
				m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).Times(0)

				// when
				resp, err := plg.createTransfer(ctx, paymentInitiation)

				// Then
				assertTransferErrorResponse(
					resp,
					err,
					"amount is required in transfer/payout request",
				)
			})

			It("Missing asset", func(ctx SpecContext) {
				// given
				paymentInitiation.Asset = ""
				m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).Times(0)

				// when
				resp, err := plg.createTransfer(ctx, paymentInitiation)

				// Then
				assertTransferErrorResponse(
					resp,
					err,
					"asset is required in transfer/payout request",
				)
			})

			DescribeTable("Missing account",
				func(ctx SpecContext, accountType string) {
					// Given a good request, but wiht missing account
					if accountType == "source" {
						paymentInitiation.SourceAccount = nil
					} else {
						paymentInitiation.DestinationAccount = nil
					}

					// Then
					m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).Times(0)

					// When
					resp, err := plg.createTransfer(ctx, paymentInitiation)

					// Then
					assertTransferErrorResponse(
						resp,
						err,
						fmt.Sprintf("%v account is required in transfer/payout request", accountType),
					)
				},
				Entry("Source account", "source"),
				Entry("Destination account", "destination"),
			)

			DescribeTable("Invalid account",
				func(ctx SpecContext, accountType string) {
					// Given a good request, but wiht missing account
					if accountType == "source" {
						paymentInitiation.SourceAccount.Metadata["bank_account_iban"] = ""
					} else {
						paymentInitiation.DestinationAccount.Metadata["bank_account_iban"] = ""
					}

					// Then
					m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).Times(0)

					// When
					resp, err := plg.createTransfer(ctx, paymentInitiation)

					// Then
					assertTransferErrorResponse(
						resp,
						err,
						fmt.Sprintf("iban is required in %v account", accountType),
					)
				},
				Entry("Source account", "source"),
				Entry("Destination account", "destination"),
			)
			It("returns error on invalid asset", func(ctx SpecContext) {
				// Given
				paymentInitiation.Asset = "EUR"
				m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).Times(0)

				// When
				resp, err := plg.createTransfer(ctx, paymentInitiation)

				// Then
				assertTransferErrorResponse(
					resp,
					err,
					"failed to get currency and precision from asset: invalid asset: EUR",
				)
			})
		})

		It("returns error when client fails", func(ctx SpecContext) {
			// Given a valid request but the client fails

			// Then
			m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).Return(
				nil,
				errors.New(":boom: oopsy"),
			)

			// When
			resp, err := plg.createTransfer(ctx, paymentInitiation)

			// Then
			assertTransferErrorResponse(resp, err, ":boom: oopsy")
		})

		Describe("Invalid response cases", func() {
			It("invalid createdAt returned", func(ctx SpecContext) {
				// Given a return with an invalid date
				transferResponse.CreatedDate = "invalid-date"

				m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).
					Times(1).
					Return(&transferResponse, nil)

				// when
				resp, err := plg.createTransfer(ctx, paymentInitiation)

				// then
				assertTransferErrorResponse(
					resp,
					err,
					"invalid time format for transfer",
				)
			})

			It("invalid amountCents returned", func(ctx SpecContext) {
				// Given a return with an invalid date
				transferResponse.AmountCents = "toto"

				m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).
					Times(1).
					Return(&transferResponse, nil)

				// when
				resp, err := plg.createTransfer(ctx, paymentInitiation)

				// then
				assertTransferErrorResponse(
					resp,
					err,
					"failed to marshal transfer: json",
				)
			})

			It("invalid status returned", func(ctx SpecContext) {
				// Given a return with an invalid date
				transferResponse.Status = "toto"

				m.EXPECT().CreateInternalTransfer(gomock.Any(), paymentInitiation.Reference, gomock.Any()).
					Times(1).
					Return(&transferResponse, nil)

				// when
				resp, err := plg.createTransfer(ctx, paymentInitiation)

				// then
				assertTransferErrorResponse(
					resp,
					err,
					fmt.Sprintf("Unexpected status on newly created transfer: %s", transferResponse.Status),
				)
			})
		})
	})
})

func assertTransferErrorResponse(resp *models.PSPPayment, err error, expectedError string) {
	Expect(err).ToNot(BeNil())
	Expect(err).To(MatchError(ContainSubstring(expectedError)))
	Expect(resp).To(BeNil())
}
