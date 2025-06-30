package moov

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Moov Payouts", func() {
	var (
		plg               *Plugin
		mockCtrl          *gomock.Controller
		mockClient        *client.MockClient
		sourceAccountID   string
		destAccountID     string
		sampleTransfer    moov.Transfer
		samplePaymentInit models.PSPPaymentInitiation
		ctx               context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = client.NewMockClient(mockCtrl)
		plg = &Plugin{
			name:   "moov",
			client: mockClient,
		}

		sourceAccountID = "source-account-123"
		destAccountID = "dest-account-456"

		// Setup sample transfer (payout response)
		sampleTransfer = moov.Transfer{
			TransferID: "transfer123",
			Status:     moov.TransferStatus_Completed,
			Amount: moov.Amount{
				Value:    1000,
				Currency: "USD",
			},
			Source: moov.TransferSource{
				PaymentMethodID:   "src-payment-method-123",
				PaymentMethodType: "ach-debit",
				Account: moov.TransferAccount{
					AccountID:   sourceAccountID,
					Email:       "source@example.com",
					DisplayName: "Source Account",
				},
				Wallet: &moov.WalletPaymentMethod{
					WalletID: "wallet-source-123",
				},
			},
			Destination: moov.TransferDestination{
				PaymentMethodID:   "dst-payment-method-456",
				PaymentMethodType: "ach-credit",
				Account: moov.TransferAccount{
					AccountID:   destAccountID,
					Email:       "destination@example.com",
					DisplayName: "Destination Account",
				},
				Wallet: &moov.WalletPaymentMethod{
					WalletID: "wallet-destination-456",
				},
			},
			CreatedOn: time.Now().UTC(),
		}

		// Setup sample payment initiation
		samplePaymentInit = models.PSPPaymentInitiation{
			SourceAccount: &models.PSPAccount{
				Reference: "source-ref-123",
				Metadata: map[string]string{
					client.MoovAccountIDMetadataKey: sourceAccountID,
				},
			},
			DestinationAccount: &models.PSPAccount{
				Reference: "dest-ref-456",
				Metadata: map[string]string{
					client.MoovAccountIDMetadataKey: destAccountID,
				},
			},
			Amount:      big.NewInt(1000),
			Asset:       "USD/2",
			Description: "Test payout",
			Metadata: map[string]string{
				client.MoovSourcePaymentMethodIDMetadataKey:      "src-payment-method-123",
				client.MoovDestinationPaymentMethodIDMetadataKey: "dst-payment-method-456",
				client.MoovPaymentTypeMetadataKey:                "ach",
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("CreatePayout", func() {
		It("should successfully create a payout", func() {
			mockClient.EXPECT().InitiatePayout(
				gomock.Any(),
				sourceAccountID,
				destAccountID,
				gomock.Any(),
			).Return(&sampleTransfer, nil)

			response, err := plg.createPayout(ctx, samplePaymentInit)

			Expect(err).To(BeNil())
			Expect(response.Payment).NotTo(BeNil())
			Expect(response.Payment.Reference).To(Equal(sampleTransfer.TransferID))
			Expect(response.Payment.Amount.String()).To(Equal(big.NewInt(1000).String()))
			Expect(response.Payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
			Expect(*response.Payment.SourceAccountReference).To(Equal("source-ref-123"))
			Expect(*response.Payment.DestinationAccountReference).To(Equal("dest-ref-456"))
			Expect(response.Payment.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))
		})

		It("should return validation error when payment initiation is invalid", func() {
			// Missing source account
			invalidPi := samplePaymentInit
			invalidPi.SourceAccount = nil

			response, err := plg.createPayout(ctx, invalidPi)

			Expect(err).To(MatchError(models.NewConnectorValidationError("sourceAccount", ErrMissingSourceAccount)))
			Expect(response.Payment).To(BeNil())
		})

		It("should return error when currency extraction fails", func() {
			// Invalid asset format
			invalidPi := samplePaymentInit
			invalidPi.Asset = "invalid-asset"

			response, err := plg.createPayout(ctx, invalidPi)

			Expect(err).NotTo(BeNil())
			Expect(response.Payment).To(BeNil())
		})

		It("should return error when InitiatePayout fails", func() {
			mockClient.EXPECT().InitiatePayout(
				gomock.Any(),
				sourceAccountID,
				destAccountID,
				gomock.Any(),
			).Return(nil, errors.New("payout failed"))

			response, err := plg.createPayout(ctx, samplePaymentInit)

			Expect(err).To(MatchError("payout failed"))
			Expect(response.Payment).To(BeNil())
		})
	})

	It("should convert a Moov transfer to a PSP payment", func() {
		payment, err := payoutToPayment(&sampleTransfer, sourceAccountID, destAccountID)

		Expect(err).To(BeNil())
		Expect(payment.Reference).To(Equal(sampleTransfer.TransferID))
		Expect(payment.Amount.String()).To(Equal(big.NewInt(1000).String()))
		Expect(payment.Asset).To(Equal("USD/2"))
		Expect(payment.Status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
		Expect(*payment.SourceAccountReference).To(Equal(sourceAccountID))
		Expect(*payment.DestinationAccountReference).To(Equal(destAccountID))
		Expect(payment.Type).To(Equal(models.PAYMENT_TYPE_TRANSFER))

		var transfer moov.Transfer
		err = json.Unmarshal(payment.Raw, &transfer)
		Expect(err).To(BeNil())
		Expect(transfer.TransferID).To(Equal(sampleTransfer.TransferID))
	})

	It("should return error when source account is missing", func() {
		pi := samplePaymentInit
		pi.SourceAccount = nil

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError("sourceAccount", ErrMissingSourceAccount)))
	})

	It("should return error when destination account is missing", func() {
		pi := samplePaymentInit
		pi.DestinationAccount = nil

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError("destinationAccount", ErrMissingDestinationAccount)))
	})

	It("should return error when amount is missing", func() {
		pi := samplePaymentInit
		pi.Amount = nil

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError("amount", ErrMissingAmount)))
	})

	It("should return error when asset is missing", func() {
		pi := samplePaymentInit
		pi.Asset = ""

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError("asset", ErrMissingAsset)))
	})

	It("should return error when destination payment method ID is missing", func() {
		pi := samplePaymentInit
		delete(pi.Metadata, client.MoovDestinationPaymentMethodIDMetadataKey)

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovDestinationPaymentMethodIDMetadataKey, ErrMissingDestinationPaymentMethodID)))
	})

	It("should return error when source payment method ID is missing", func() {
		pi := samplePaymentInit
		delete(pi.Metadata, client.MoovSourcePaymentMethodIDMetadataKey)

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovSourcePaymentMethodIDMetadataKey, ErrMissingSourcePaymentMethodID)))
	})

	It("should return error when sales tax currency is provided but value is not", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovSalesTaxAmountCurrencyMetadataKey] = "USD"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovSalesTaxAmountValueMetadataKey, ErrMissingSalesTaxValue)))
	})

	It("should return error when sales tax value is provided but currency is not", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovSalesTaxAmountValueMetadataKey] = "100"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovSalesTaxAmountCurrencyMetadataKey, ErrMissingSalesTaxCurrency)))
	})

	It("should allow facilitator fee with only markup (no total)", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeMarkupMetadataKey] = "100"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow facilitator fee with only total (no markup)", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeTotalMetadataKey] = "200"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should return error when both markup and total fee structures are provided", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeMarkupMetadataKey] = "100"
		pi.Metadata[client.MoovFacilitatorFeeTotalMetadataKey] = "200"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovFacilitatorFeeTotalMetadataKey, ErrConflictingFacilitatorFeeStructures)))
	})

	It("should return error when both markup and markupDecimal are provided", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeMarkupMetadataKey] = "100"
		pi.Metadata[client.MoovFacilitatorFeeMarkupDecimalMetadataKey] = "100.50"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovFacilitatorFeeMarkupDecimalMetadataKey, ErrConflictingFacilitatorFeeMarkupFormats)))
	})

	It("should return error when both total and totalDecimal are provided", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeTotalMetadataKey] = "200"
		pi.Metadata[client.MoovFacilitatorFeeTotalDecimalMetadataKey] = "200.50"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovFacilitatorFeeTotalDecimalMetadataKey, ErrConflictingFacilitatorFeeTotalFormats)))
	})

	It("should accept valid facilitator fee with only markup", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeMarkupMetadataKey] = "100"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(BeNil())
	})

	It("should accept valid facilitator fee with only markupDecimal", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeMarkupDecimalMetadataKey] = "100.50"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(BeNil())
	})

	It("should accept valid facilitator fee with only totalDecimal", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeTotalDecimalMetadataKey] = "200.75"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(BeNil())
	})

	It("should return error when source ACH SEC code is invalid", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovPaymentTypeMetadataKey] = "ach"
		pi.Metadata[client.MoovSourceACHSecCodeMetadataKey] = "INVALID"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovSourceACHSecCodeMetadataKey, ErrInvalidSourceACHSecCode)))
	})

	It("should return error when source ACH company entry description is too long", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovPaymentTypeMetadataKey] = "ach"
		pi.Metadata[client.MoovSourceACHCompanyEntryDescriptionMetadataKey] = "ThisDescriptionIsTooLong"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovSourceACHCompanyEntryDescriptionMetadataKey, ErrSourceACHCompanyEntryDescriptionTooLong)))
	})

	It("should return error when destination ACH company entry description is too long", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovPaymentTypeMetadataKey] = "ach"
		pi.DestinationAccount.Metadata[client.MoovDestinationACHCompanyEntryDescriptionMetadataKey] = "ThisDescriptionIsTooLong"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovDestinationACHCompanyEntryDescriptionMetadataKey, ErrDestinationACHCompanyEntryDescriptionTooLong)))
	})

	It("should return error when destination ACH originating company name is too long", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovPaymentTypeMetadataKey] = "ach"
		pi.DestinationAccount.Metadata[client.MoovDestinationACHOriginatingCompanyNameMetadataKey] = "ThisCompanyNameIsFarTooLong"

		err := plg.validateTransferPayoutRequests(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovDestinationACHOriginatingCompanyNameMetadataKey, ErrDestinationACHOriginatingCompanyNameTooLong)))
	})

	It("should extract sales tax amount correctly", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovSalesTaxAmountValueMetadataKey] = "150"
		pi.Metadata[client.MoovSalesTaxAmountCurrencyMetadataKey] = "USD/2"

		salesTax, err := extractSalesTax(pi)

		Expect(err).To(BeNil())
		Expect(salesTax.Value).To(Equal(int64(150)))
		Expect(salesTax.Currency).To(Equal("USD"))
	})

	It("should return empty sales tax when not provided", func() {
		pi := samplePaymentInit
		// No sales tax metadata

		salesTax, err := extractSalesTax(pi)

		Expect(err).To(BeNil())
		Expect(salesTax).To(BeNil())
	})

	It("should return error when sales tax amount is invalid", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovSalesTaxAmountValueMetadataKey] = "not-a-number"
		pi.Metadata[client.MoovSalesTaxAmountCurrencyMetadataKey] = "USD"

		_, err := extractSalesTax(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovSalesTaxAmountValueMetadataKey, ErrInvalidSalesTaxAmount)))
	})

	It("should extract facilitator fee correctly", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeTotalMetadataKey] = "200"
		pi.Metadata[client.MoovFacilitatorFeeTotalDecimalMetadataKey] = "200"
		pi.Metadata[client.MoovFacilitatorFeeMarkupDecimalMetadataKey] = "50"

		fee, err := extractFacilitatorFee(pi)

		Expect(err).To(BeNil())
		Expect(*fee.Total).To(Equal(int64(200)))
		Expect(*fee.TotalDecimal).To(Equal("200"))
		Expect(*fee.Markup).To(Equal(int64(50)))
		Expect(*fee.MarkupDecimal).To(Equal("50"))
	})

	It("should return empty facilitator fee when not provided", func() {
		pi := samplePaymentInit
		// No facilitator fee metadata

		fee, err := extractFacilitatorFee(pi)

		Expect(err).To(BeNil())
		Expect(fee.Total).To(BeNil())
		Expect(fee.TotalDecimal).To(BeNil())
		Expect(fee.Markup).To(BeNil())
		Expect(fee.MarkupDecimal).To(BeNil())
	})

	It("should return error when facilitator fee total is invalid", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeTotalMetadataKey] = "not-a-number"

		_, err := extractFacilitatorFee(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovFacilitatorFeeTotalMetadataKey, ErrInvalidFacilitatorFeeTotal)))
	})

	It("should return error when facilitator fee markup is invalid", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovFacilitatorFeeTotalDecimalMetadataKey] = "200"
		pi.Metadata[client.MoovFacilitatorFeeMarkupDecimalMetadataKey] = "not-a-number"

		_, err := extractFacilitatorFee(pi)

		Expect(err).To(MatchError(models.NewConnectorValidationError(client.MoovFacilitatorFeeMarkupDecimalMetadataKey, ErrInvalidFacilitatorFeeMarkup)))
	})

	It("should extract payment source correctly", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovSourcePaymentMethodIDMetadataKey] = "src-payment-method-123"
		pi.Metadata[client.MoovSourceACHCompanyEntryDescriptionMetadataKey] = "ACME Corp"
		pi.Metadata[client.MoovSourceACHSecCodeMetadataKey] = "PPD"
		pi.Metadata[client.MoovSourceTransferIDMetadataKey] = "transfer123"

		source := extractPaymentSource(pi)

		Expect(source.AchDetails.CompanyEntryDescription).To(Equal("ACME Corp"))
		Expect(string(*source.AchDetails.SecCode)).To(Equal("PPD"))
		Expect(source.TransferID).To(Equal("transfer123"))
	})

	It("should extract payment destination correctly", func() {
		pi := samplePaymentInit
		pi.Metadata[client.MoovDestinationPaymentMethodIDMetadataKey] = "dst-payment-method-456"
		pi.Metadata[client.MoovDestinationACHCompanyEntryDescriptionMetadataKey] = "ACME Inc"
		pi.Metadata[client.MoovDestinationACHOriginatingCompanyNameMetadataKey] = "ACME"

		destination := extractPaymentDestination(pi)

		Expect(destination.PaymentMethodID).To(Equal("dst-payment-method-456"))
		Expect(destination.AchDetails.CompanyEntryDescription).To(Equal("ACME Inc"))
		Expect(destination.AchDetails.OriginatingCompanyName).To(Equal("ACME"))
	})

	It("should map Completed status to SUCCEEDED", func() {
		status := mapStatus(moov.TransferStatus_Completed)
		Expect(status).To(Equal(models.PAYMENT_STATUS_SUCCEEDED))
	})

	It("should map Failed status to FAILED", func() {
		status := mapStatus(moov.TransferStatus_Failed)
		Expect(status).To(Equal(models.PAYMENT_STATUS_FAILED))
	})

	It("should map Canceled status to CANCELLED", func() {
		status := mapStatus(moov.TransferStatus_Canceled)
		Expect(status).To(Equal(models.PAYMENT_STATUS_CANCELLED))
	})

	It("should map Pending status to PENDING", func() {
		status := mapStatus(moov.TransferStatus_Pending)
		Expect(status).To(Equal(models.PAYMENT_STATUS_PENDING))
	})

	It("should map Created status to PENDING", func() {
		status := mapStatus(moov.TransferStatus_Created)
		Expect(status).To(Equal(models.PAYMENT_STATUS_PENDING))
	})

	It("should map Reversed status to REFUNDED", func() {
		status := mapStatus(moov.TransferStatus_Reversed)
		Expect(status).To(Equal(models.PAYMENT_STATUS_REFUNDED))
	})

	It("should map Queued status to PENDING", func() {
		status := mapStatus(moov.TransferStatus_Queued)
		Expect(status).To(Equal(models.PAYMENT_STATUS_PENDING))
	})

	It("should map unknown status to UNKNOWN", func() {
		status := mapStatus("unknown")
		Expect(status).To(Equal(models.PAYMENT_STATUS_UNKNOWN))
	})

	It("should map transfer between two wallets to TRANSFER", func() {
		transfer := sampleTransfer

		paymentType := mapPaymentType(transfer)

		Expect(paymentType).To(Equal(models.PAYMENT_TYPE_TRANSFER))
	})

	It("should map payout from wallet to PAYOUT", func() {
		transfer := sampleTransfer
		transfer.Destination.Wallet = nil

		paymentType := mapPaymentType(transfer)

		Expect(paymentType).To(Equal(models.PAYMENT_TYPE_PAYOUT))
	})

	It("should map payin to wallet to PAYIN", func() {
		transfer := sampleTransfer
		transfer.Source.Wallet = nil

		paymentType := mapPaymentType(transfer)

		Expect(paymentType).To(Equal(models.PAYMENT_TYPE_PAYIN))
	})

	It("should map bank transfer to TRANSFER", func() {
		transfer := sampleTransfer
		transfer.Source.Wallet = nil
		transfer.Destination.Wallet = nil
		transfer.Source.BankAccount = &moov.BankAccountPaymentMethod{
			BankAccountID: "bank-123",
		}
		transfer.Destination.BankAccount = &moov.BankAccountPaymentMethod{
			BankAccountID: "bank-456",
		}

		paymentType := mapPaymentType(transfer)

		Expect(paymentType).To(Equal(models.PAYMENT_TYPE_PAYOUT))
	})

	Context("create payout with moov client", func() {
		var (
			mockedService *client.MockMoovClient
		)

		BeforeEach((func() {
			ctrl := gomock.NewController(GinkgoT())
			mockedService = client.NewMockMoovClient(ctrl)

			plg.client, _ = client.New("moov", "https://example.com", "access_token", "test", "test")
			plg.client.NewWithClient(mockedService)
		}))

		It("should create a payout", func() {
			mockedService.EXPECT().GetMoovTransferOptions(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("moov transfer options error"))

			response, err := plg.client.InitiatePayout(ctx, sourceAccountID, destAccountID, moov.CreateTransfer{})

			Expect(err).To(MatchError(ContainSubstring("failed to get transfer options: moov transfer options error")))
			Expect(response).To(BeNil())
		})

		It("should return error when GetMoovTransferOptions fails", func() {
			mockedService.EXPECT().GetMoovTransferOptions(gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("moov transfer options error"))

			response, err := plg.client.InitiatePayout(ctx, sourceAccountID, destAccountID, moov.CreateTransfer{})

			Expect(err).To(MatchError(ContainSubstring("failed to get transfer options: moov transfer options error")))
			Expect(response).To(BeNil())
		})

		It("should return error when no source options are found", func() {
			// Create transfer options with empty source options
			transferOptions := &moov.TransferOptions{
				SourceOptions: []moov.PaymentMethod{}, // Empty source options
				DestinationOptions: []moov.PaymentMethod{
					{
						PaymentMethodID:   "dst-payment-method-456",
						PaymentMethodType: "ach-credit-standard",
					},
				},
			}

			mockedService.EXPECT().GetMoovTransferOptions(gomock.Any(), gomock.Any()).
				Return(transferOptions, nil)

			response, err := plg.client.InitiatePayout(ctx, sourceAccountID, destAccountID, moov.CreateTransfer{})

			Expect(err).To(MatchError(models.NewConnectorValidationError("SourceAccountID", errors.New("no source options found in Moov for source account"))))
			Expect(response).To(BeNil())
		})

		It("should return error when no destination options are found", func() {
			// Create transfer options with empty destination options
			transferOptions := &moov.TransferOptions{
				SourceOptions: []moov.PaymentMethod{
					{
						PaymentMethodID:   "src-payment-method-123",
						PaymentMethodType: "moov-wallet",
					},
				},
				DestinationOptions: []moov.PaymentMethod{}, // Empty destination options
			}

			mockedService.EXPECT().GetMoovTransferOptions(gomock.Any(), gomock.Any()).
				Return(transferOptions, nil)

			response, err := plg.client.InitiatePayout(ctx, sourceAccountID, destAccountID, moov.CreateTransfer{})

			Expect(err).To(MatchError(models.NewConnectorValidationError("DestinationAccountID", errors.New("no destination options found in Moov for destination account"))))
			Expect(response).To(BeNil())
		})

		It("should return error when source payment method is not found", func() {
			// Create transfer options with different source payment method ID than requested
			transferOptions := &moov.TransferOptions{
				SourceOptions: []moov.PaymentMethod{
					{
						PaymentMethodID:   "different-source-id", // Different than what will be in the request
						PaymentMethodType: "moov-wallet",
					},
				},
				DestinationOptions: []moov.PaymentMethod{
					{
						PaymentMethodID:   "dst-payment-method-456",
						PaymentMethodType: "ach-credit-standard",
					},
				},
			}

			request := moov.CreateTransfer{
				Source: moov.CreateTransfer_Source{
					PaymentMethodID: "src-payment-method-123", // This doesn't match what's in transfer options
				},
				Destination: moov.CreateTransfer_Destination{
					PaymentMethodID: "dst-payment-method-456",
				},
			}

			mockedService.EXPECT().GetMoovTransferOptions(gomock.Any(), gomock.Any()).
				Return(transferOptions, nil)

			response, err := plg.client.InitiatePayout(ctx, sourceAccountID, destAccountID, request)

			Expect(err).To(MatchError(models.NewConnectorValidationError("SourceAccountID", errors.New("source payment method src-payment-method-123 not found in available transfer options"))))
			Expect(response).To(BeNil())
		})

		It("should return error when destination payment method is not found", func() {
			// Create transfer options with different destination payment method ID than requested
			transferOptions := &moov.TransferOptions{
				SourceOptions: []moov.PaymentMethod{
					{
						PaymentMethodID:   "src-payment-method-123",
						PaymentMethodType: "moov-wallet",
					},
				},
				DestinationOptions: []moov.PaymentMethod{
					{
						PaymentMethodID:   "different-dest-id", // Different than what will be in the request
						PaymentMethodType: "ach-credit-standard",
					},
				},
			}

			request := moov.CreateTransfer{
				Source: moov.CreateTransfer_Source{
					PaymentMethodID: "src-payment-method-123",
				},
				Destination: moov.CreateTransfer_Destination{
					PaymentMethodID: "dst-payment-method-456", // This doesn't match what's in transfer options
				},
			}

			mockedService.EXPECT().GetMoovTransferOptions(gomock.Any(), gomock.Any()).
				Return(transferOptions, nil)

			response, err := plg.client.InitiatePayout(ctx, sourceAccountID, destAccountID, request)

			Expect(err).To(MatchError(models.NewConnectorValidationError("DestinationAccountID", errors.New("destination payment method dst-payment-method-456 not found in available transfer options"))))
			Expect(response).To(BeNil())
		})

	})
})
