package column

import (
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Column Plugin Payments", func() {
	var (
		plg *Plugin
	)

	BeforeEach(func() {
		plg = &Plugin{}
	})

	Context("fetching next payments", func() {
		var (
			mockHTTPClient *client.MockHTTPClient
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockHTTPClient = client.NewMockHTTPClient(ctrl)
			plg.client = client.New("test", "aseplye", "https://test.com")
			plg.client.SetHttpClient(mockHTTPClient)
		})

		Context("validateTransferRequest", func() {
			It("should validate a valid transfer request", func() {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnAllowOverdraftMetadataKey:          "true",
						client.ColumnHoldMetadataKey:                    "false",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				err := plg.validateTransferRequest(pi)
				Expect(err).To(BeNil())
			})

			It("should return error when amount is missing", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Asset: "USD/2",
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field amount must be provided"))
			})

			It("should return error when source account reference is missing", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount:   big.NewInt(100),
					Asset:    "USD",
					Metadata: map[string]string{},
					SourceAccount: &models.PSPAccount{
						Name:     pointer.For("Test Source"),
						Metadata: map[string]string{},
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required sourceAccount field reference is missing"))
			})

			It("should return error when address line 1 is provided but city is missing", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnAddressLine1MetadataKey:            "123 Main St",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError(fmt.Sprintf("required metadata field %s is missing", client.ColumnCityMetadataKey)))
			})

			It("should return error when asset is missing", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field asset must be provided"))
			})

			It("should return error when metadata is nil", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD/2",
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field metadata must be provided"))
			})

			It("should return error when source account is nil", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount:   big.NewInt(100),
					Asset:    "USD",
					Metadata: map[string]string{},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field sourceAccount is missing"))
			})

			It("should return error when source account name is nil", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount:        big.NewInt(100),
					Asset:         "USD",
					Metadata:      map[string]string{},
					SourceAccount: &models.PSPAccount{},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required sourceAccount field name is missing"))
			})

			It("should return error when destination account is nil", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD/2",
					Metadata: map[string]string{
						client.ColumnSenderAccountNumberIdMetadataKey: "src_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field destinationAccount must be provided"))
			})

			It("should return error when destination account reference is missing", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnSenderAccountNumberIdMetadataKey: "src_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required destinationAccount field reference is missing"))
			})

			It("should return error when allow overdraft is invalid", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnAllowOverdraftMetadataKey:          "invalid",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field allow overdraft must be provided"))
			})

			It("should return error when hold is invalid", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnHoldMetadataKey:                    "invalid",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError("required field hold must be provided"))
			})

			It("should return error when country code is missing with address line 1", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnAddressLine1MetadataKey:            "123 Main St",
						client.ColumnCityMetadataKey:                    "New York",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError(fmt.Sprintf("required metadata field %s is missing", client.ColumnCountryCodeMetadataKey)))
			})

			It("should return error when city is provided without address line 1", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnCityMetadataKey:                    "New York",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnCityMetadataKey)))
			})

			It("should return error when state is provided without address line 1", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnStateMetadataKey:                   "NY",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnStateMetadataKey)))
			})

			It("should return error when postal code is provided without address line 1", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnAddressPostalCodeMetadataKey:       "10001",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnAddressPostalCodeMetadataKey)))
			})

			It("should return error when country code is provided without address line 1", func(ctx SpecContext) {
				pi := models.PSPPaymentInitiation{
					Amount: big.NewInt(100),
					Asset:  "USD",
					Metadata: map[string]string{
						client.ColumnCountryCodeMetadataKey:             "US",
						client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
						client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
					},
					SourceAccount: &models.PSPAccount{
						Name:      pointer.For("Test Source"),
						Reference: "src_ref",
					},
					DestinationAccount: &models.PSPAccount{
						Reference: "dst_ref",
					},
				}

				_, err := plg.CreateTransfer(ctx, models.CreateTransferRequest{
					PaymentInitiation: pi,
				})
				Expect(err).To(MatchError(fmt.Sprintf("metadata field %s is not required when addressLine1 is not provided", client.ColumnCountryCodeMetadataKey)))
			})

		})

		Context("HTTP Request Creation Errors", func() {
			BeforeEach(func() {
				ctrl := gomock.NewController(GinkgoT())
				mockHTTPClient = client.NewMockHTTPClient(ctrl)
				plg.client = client.New("test", "aseplye", "http://invalid:port")
				plg.client.SetHttpClient(mockHTTPClient)
			})

			It("should return an error when transfer request URL is invalid", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						Metadata: map[string]string{
							client.ColumnAllowOverdraftMetadataKey:          "true",
							client.ColumnHoldMetadataKey:                    "false",
							client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
							client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "dst_ref",
						},
					},
				}

				res, err := plg.CreateTransfer(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreateTransferResponse{}))
				Expect(err.Error()).To(ContainSubstring("failed to create payments request"))
			})
		})

		Context("Column API Errors", func() {
			BeforeEach(func() {
				ctrl := gomock.NewController(GinkgoT())
				mockHTTPClient = client.NewMockHTTPClient(ctrl)
				plg.client = client.New("test", "aseplye", "https://test.com")
				plg.client.SetHttpClient(mockHTTPClient)
			})

			It("should return an error when HTTP client Do fails", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						Metadata: map[string]string{
							client.ColumnAllowOverdraftMetadataKey:          "true",
							client.ColumnHoldMetadataKey:                    "false",
							client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
							client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "dst_ref",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					500,
					fmt.Errorf("mock HTTP client error"),
				)

				res, err := plg.CreateTransfer(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreateTransferResponse{}))
				Expect(err.Error()).To(ContainSubstring("mock HTTP client error"))
			})

			It("should return an error when asset currency is invalid", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "INVALID/2",
						Metadata: map[string]string{
							client.ColumnAllowOverdraftMetadataKey:          "true",
							client.ColumnHoldMetadataKey:                    "false",
							client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
							client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "dst_ref",
						},
					},
				}

				res, err := plg.CreateTransfer(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreateTransferResponse{}))
				Expect(err.Error()).To(ContainSubstring("failed to get currency and precision from asset"))
			})

			It("should return an error when currency is not supported", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						Metadata: map[string]string{
							client.ColumnAllowOverdraftMetadataKey:          "true",
							client.ColumnHoldMetadataKey:                    "false",
							client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
							client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "dst_ref",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.TransferResponse{
					CurrencyCode: "UNSUPPORTED",
					CreatedAt:    "2021-01-01T00:00:00Z",
					ID:           "test-id",
					Amount:       100,
				})

				res, err := plg.CreateTransfer(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreateTransferResponse{}))
				Expect(err.Error()).To(ContainSubstring("unsupported currency: UNSUPPORTED"))
			})

			It("should return an error when parsing invalid CreatedAt timestamp", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						Metadata: map[string]string{
							client.ColumnAllowOverdraftMetadataKey:          "true",
							client.ColumnHoldMetadataKey:                    "false",
							client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
							client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "dst_ref",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.TransferResponse{
					CreatedAt: "invalid-timestamp",
					ID:        "test-id",
					Amount:    100,
				})

				res, err := plg.CreateTransfer(ctx, req)
				Expect(err).ToNot(BeNil())
				Expect(res).To(Equal(models.CreateTransferResponse{}))
				Expect(err.Error()).To(ContainSubstring("failed to parse posted date"))
			})

			It("should successfully create a transfer", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						Metadata: map[string]string{
							client.ColumnAllowOverdraftMetadataKey:          "true",
							client.ColumnHoldMetadataKey:                    "false",
							client.ColumnSenderAccountNumberIdMetadataKey:   "src_acc_num",
							client.ColumnReceiverAccountNumberIdMetadataKey: "dst_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
						DestinationAccount: &models.PSPAccount{
							Reference: "dst_ref",
						},
					},
				}

				mockHTTPClient.EXPECT().Do(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(
					200,
					nil,
				).SetArg(2, client.TransferResponse{
					CreatedAt:    "2024-03-20T10:00:00Z",
					CurrencyCode: "USD",
					ID:           "test-transfer-id",
					Amount:       100,
				})

				res, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(BeNil())
				Expect(res.Payment).ToNot(BeNil())
				Expect(res.Payment.Reference).To(Equal("test-transfer-id"))
				Expect(res.Payment.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
				Expect(res.Payment.Asset).To(Equal("USD/2"))
				Expect(res.PollingTransferID).To(BeNil())
			})
		})
	})
})
