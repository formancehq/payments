package column

import (
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Column Plugin Payments", func() {
	var (
		ctrl           *gomock.Controller
		mockHTTPClient *client.MockHTTPClient
		plg            models.Plugin
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHTTPClient = client.NewMockHTTPClient(ctrl)
		c := client.New("test", "aseplye", "https://test.com")
		c.SetHttpClient(mockHTTPClient)
		plg = &Plugin{client: c}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("fetching next payments", func() {
		Context("validateTransferRequest", func() {
			It("should return error when amount is missing", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Asset: "USD/2",
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingAmount.Error()))
			})

			It("should return error when source account reference is missing", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount:   big.NewInt(100),
						Asset:    "USD",
						Metadata: map[string]string{},
						SourceAccount: &models.PSPAccount{
							Name:     pointer.For("Test Source"),
							Metadata: map[string]string{},
						},
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrSourceAccountReferenceRequired.Error()))
			})

			It("should return error when address line 1 is provided but city is missing", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
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
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(MatchError("validation error occurred for field com.column.spec/address_city: required metadata field com.column.spec/address_city must be provided"))
			})

			It("should return error when asset is missing", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingAsset.Error()))
			})

			It("should return error when metadata is nil", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingMetadata.Error()))
			})

			It("should return error when source account is nil", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount:   big.NewInt(100),
						Asset:    "USD",
						Metadata: map[string]string{},
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingSourceAccount.Error()))
			})

			It("should return error when source account name is nil", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount:        big.NewInt(100),
						Asset:         "USD",
						Metadata:      map[string]string{},
						SourceAccount: &models.PSPAccount{},
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingSourceAccountName.Error()))
			})

			It("should return error when destination account is nil", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD/2",
						Metadata: map[string]string{
							client.ColumnSenderAccountNumberIdMetadataKey: "src_acc_num",
						},
						SourceAccount: &models.PSPAccount{
							Name:      pointer.For("Test Source"),
							Reference: "src_ref",
						},
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingDestinationAccount.Error()))
			})

			It("should return error when destination account reference is missing", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
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
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingDestinationAccountReference.Error()))
			})

			It("should return error when allow overdraft is invalid", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
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
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingMetadataAllowOverDrafts.Error()))
			})

			It("should return error when hold is invalid", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
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
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err.Error()).To(ContainSubstring(ErrMissingMetadataHold.Error()))
			})

			It("should return error when country code is missing with address line 1", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD",
						Metadata: map[string]string{
							client.ColumnAddressLine1MetadataKey:            "123 Main St",
							client.ColumnAddressCityMetadataKey:             "New York",
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

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(MatchError("validation error occurred for field com.column.spec/address_country_code: required metadata field com.column.spec/address_country_code must be provided"))
			})

			It("should return error when city is provided without address line 1", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD",
						Metadata: map[string]string{
							client.ColumnAddressCityMetadataKey:             "New York",
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

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(MatchError("validation error occurred for field com.column.spec/address_city: metadata field com.column.spec/address_city is not required when addressLine1 is not provided"))
			})

			It("should return error when state is provided without address line 1", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD",
						Metadata: map[string]string{
							client.ColumnAddressStateMetadataKey:            "NY",
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

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(MatchError("validation error occurred for field com.column.spec/address_state: metadata field com.column.spec/address_state is not required when addressLine1 is not provided"))
			})

			It("should return error when postal code is provided without address line 1", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
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
					},
				}

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(MatchError("validation error occurred for field com.column.spec/address_postal_code: metadata field com.column.spec/address_postal_code is not required when addressLine1 is not provided"))
			})

			It("should return error when country code is provided without address line 1", func(ctx SpecContext) {
				req := models.CreateTransferRequest{
					PaymentInitiation: models.PSPPaymentInitiation{
						Amount: big.NewInt(100),
						Asset:  "USD",
						Metadata: map[string]string{
							client.ColumnAddressCountryCodeMetadataKey:      "US",
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

				_, err := plg.CreateTransfer(ctx, req)
				Expect(err).To(MatchError("validation error occurred for field com.column.spec/address_country_code: metadata field com.column.spec/address_country_code is not required when addressLine1 is not provided"))
			})

		})

		Context("HTTP Request Creation Errors", func() {
			var (
				plg models.Plugin
			)
			BeforeEach(func() {
				c := client.New("test", "aseplye", "http://invalid:port")
				c.SetHttpClient(mockHTTPClient)
				plg = &Plugin{client: c}
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
				Expect(err.Error()).To(ContainSubstring("failed to create request: parse"))
			})
		})

		Context("Column API Errors", func() {
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
