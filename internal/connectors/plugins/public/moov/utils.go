package moov

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (p *Plugin) validateTransferPayoutRequests(pi models.PSPPaymentInitiation) error {
	// Validate required fields
	if pi.SourceAccount == nil {
		return models.NewConnectorValidationError("sourceAccount", ErrMissingSourceAccount)
	}

	if pi.DestinationAccount == nil {
		return models.NewConnectorValidationError("destinationAccount", ErrMissingDestinationAccount)
	}

	if pi.Amount == nil {
		return models.NewConnectorValidationError("amount", ErrMissingAmount)
	}

	if pi.Asset == "" {
		return models.NewConnectorValidationError("asset", ErrMissingAsset)
	}

	// Validate destination payment method
	destinationPaymentMethodId := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovDestinationPaymentMethodIDMetadataKey)
	if destinationPaymentMethodId == "" {
		return models.NewConnectorValidationError(client.MoovDestinationPaymentMethodIDMetadataKey, ErrMissingDestinationPaymentMethodID)
	}

	// Validate source payment method
	sourcePaymentMethodId := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourcePaymentMethodIDMetadataKey)
	if sourcePaymentMethodId == "" {
		return models.NewConnectorValidationError(client.MoovSourcePaymentMethodIDMetadataKey, ErrMissingSourcePaymentMethodID)
	}

	// Validate sales tax amount - if currency is provided, value must be provided
	salesTaxCurrency := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSalesTaxAmountCurrencyMetadataKey)
	salesTaxValue := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSalesTaxAmountValueMetadataKey)
	if salesTaxCurrency != "" && salesTaxValue == "" {
		return models.NewConnectorValidationError(client.MoovSalesTaxAmountValueMetadataKey, ErrMissingSalesTaxValue)
	}
	if salesTaxValue != "" && salesTaxCurrency == "" {
		return models.NewConnectorValidationError(client.MoovSalesTaxAmountCurrencyMetadataKey, ErrMissingSalesTaxCurrency)
	}

	// Validate facilitator fee - check if related fields are provided consistently
	facilitatorFeeMarkup := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeMarkupMetadataKey)
	facilitatorFeeMarkupDecimal := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeMarkupDecimalMetadataKey)
	facilitatorFeeTotal := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeTotalMetadataKey)
	facilitatorFeeTotalDecimal := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeTotalDecimalMetadataKey)

	// Check if any facilitator fee field is provided
	hasFacilitatorFee := facilitatorFeeMarkup != "" || facilitatorFeeMarkupDecimal != "" || facilitatorFeeTotal != "" || facilitatorFeeTotalDecimal != ""

	if hasFacilitatorFee {
		// According to Moov API docs: "specify your fee using either total/totalDecimal or markup/markupDecimal"
		hasMarkup := facilitatorFeeMarkup != "" || facilitatorFeeMarkupDecimal != ""
		hasTotal := facilitatorFeeTotal != "" || facilitatorFeeTotalDecimal != ""

		// Ensure only one fee structure is used (either total OR markup, not both)
		if hasMarkup && hasTotal {
			return models.NewConnectorValidationError(client.MoovFacilitatorFeeTotalMetadataKey, ErrConflictingFacilitatorFeeStructures)
		}

		// Ensure at least one fee structure is provided
		if !hasMarkup && !hasTotal {
			return models.NewConnectorValidationError(client.MoovFacilitatorFeeMarkupMetadataKey, ErrMissingFacilitatorFeeStructure)
		}

		// Ensure only one format is used for markup (either markup or markupDecimal, not both)
		if facilitatorFeeMarkup != "" && facilitatorFeeMarkupDecimal != "" {
			return models.NewConnectorValidationError(client.MoovFacilitatorFeeMarkupDecimalMetadataKey, ErrConflictingFacilitatorFeeMarkupFormats)
		}

		// Ensure only one format is used for total (either total or totalDecimal, not both)
		if facilitatorFeeTotal != "" && facilitatorFeeTotalDecimal != "" {
			return models.NewConnectorValidationError(client.MoovFacilitatorFeeTotalDecimalMetadataKey, ErrConflictingFacilitatorFeeTotalFormats)
		}
	}

	paymentType := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovPaymentTypeMetadataKey)
	// Validate source ACH fields if payment type is ACH
	if paymentType == "ach" {
		sourceACHSecCode := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourceACHSecCodeMetadataKey)
		validSecCodes := []string{"CCD", "PPD", "TEL", "WEB"}
		if sourceACHSecCode != "" && !slices.Contains(validSecCodes, sourceACHSecCode) {
			return models.NewConnectorValidationError(client.MoovSourceACHSecCodeMetadataKey, ErrInvalidSourceACHSecCode)
		}

		// Validate ACH Company Entry Description length (max 10 characters)
		sourceACHCompanyEntryDescription := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourceACHCompanyEntryDescriptionMetadataKey)
		if len(sourceACHCompanyEntryDescription) > 10 {
			return models.NewConnectorValidationError(client.MoovSourceACHCompanyEntryDescriptionMetadataKey, ErrSourceACHCompanyEntryDescriptionTooLong)
		}

		destinationACHCompanyEntryDescription := models.ExtractNamespacedMetadata(pi.DestinationAccount.Metadata, client.MoovDestinationACHCompanyEntryDescriptionMetadataKey)
		if len(destinationACHCompanyEntryDescription) > 10 {
			return models.NewConnectorValidationError(client.MoovDestinationACHCompanyEntryDescriptionMetadataKey, ErrDestinationACHCompanyEntryDescriptionTooLong)
		}

		// Validate ACH Originating Company Name length (max 16 characters)
		destinationACHOriginatingCompanyName := models.ExtractNamespacedMetadata(pi.DestinationAccount.Metadata, client.MoovDestinationACHOriginatingCompanyNameMetadataKey)
		if len(destinationACHOriginatingCompanyName) > 16 {
			return models.NewConnectorValidationError(client.MoovDestinationACHOriginatingCompanyNameMetadataKey, ErrDestinationACHOriginatingCompanyNameTooLong)
		}
	}

	return nil
}

func mapStatus(status moov.TransferStatus) models.PaymentStatus {
	switch status {
	case moov.TransferStatus_Completed:
		return models.PAYMENT_STATUS_SUCCEEDED
	case moov.TransferStatus_Failed:
		return models.PAYMENT_STATUS_FAILED
	case moov.TransferStatus_Canceled:
		return models.PAYMENT_STATUS_CANCELLED
	case moov.TransferStatus_Created, moov.TransferStatus_Pending:
		return models.PAYMENT_STATUS_PENDING
	case moov.TransferStatus_Reversed:
		return models.PAYMENT_STATUS_REFUNDED
	case moov.TransferStatus_Queued:
		return models.PAYMENT_STATUS_PENDING
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}

func mapPaymentType(transfer moov.Transfer) models.PaymentType {
	var paymentType models.PaymentType

	if transfer.Source.Wallet != nil &&
		transfer.Destination.Wallet != nil &&
		transfer.Source.Wallet.WalletID != "" &&
		transfer.Destination.Wallet.WalletID != "" {
		// This is a transfer between two Moov wallets
		paymentType = models.PAYMENT_TYPE_TRANSFER
	} else if transfer.Source.Wallet != nil && transfer.Source.Wallet.WalletID != "" {
		// This is a payout from a Moov wallet to a bank account or a card, apple pay, etc.
		paymentType = models.PAYMENT_TYPE_PAYOUT
	} else if transfer.Destination.Wallet != nil && transfer.Destination.Wallet.WalletID != "" {
		// This is a payout from a bank account or a card to a Moov wallet
		paymentType = models.PAYMENT_TYPE_PAYIN
	} else if transfer.Source.BankAccount != nil && transfer.Destination.BankAccount != nil {
		/**
		Since there are no walletIDs in either source or destination, this transaction doesn't fit directly into the three categories. It appears to be bank transfer between two bank accounts, where money is debited from one account and credited to another, without involving a Moov wallet.
		*/
		paymentType = models.PAYMENT_TYPE_PAYOUT
	} else {
		paymentType = models.PAYMENT_TYPE_UNKNOWN
	}

	return paymentType
}

func mapPaymentMetadata(transfer moov.Transfer) map[string]string {
	metadata := map[string]string{}

	metadata[client.MoovSourcePaymentMethodTypeMetadataKey] = string(transfer.Source.PaymentMethodType)
	metadata[client.MoovDestinationPaymentMethodTypeMetadataKey] = string(transfer.Destination.PaymentMethodType)

	// Source account info
	metadata[client.MoovSourceAccountEmailMetadataKey] = transfer.Source.Account.Email
	metadata[client.MoovSourceAccountDisplayNameMetadataKey] = transfer.Source.Account.DisplayName

	// Destination account info
	metadata[client.MoovDestinationAccountEmailMetadataKey] = transfer.Destination.Account.Email
	metadata[client.MoovDestinationAccountDisplayNameMetadataKey] = transfer.Destination.Account.DisplayName

	// Source bank account info
	if transfer.Source.BankAccount != nil {
		metadata[client.MoovSourceBankAccountIDMetadataKey] = transfer.Source.BankAccount.BankAccountID
		metadata[client.MoovSourceHolderNameMetadataKey] = transfer.Source.BankAccount.HolderName
		metadata[client.MoovFingerprintMetadataKey] = transfer.Source.BankAccount.Fingerprint
		metadata[client.MoovStatusMetadataKey] = string(transfer.Source.BankAccount.Status)
		metadata[client.MoovBankNameMetadataKey] = transfer.Source.BankAccount.BankName
		metadata[client.MoovRoutingNumberMetadataKey] = transfer.Source.BankAccount.RoutingNumber
		metadata[client.MoovLastFourAccountNumberMetadataKey] = transfer.Source.BankAccount.LastFourAccountNumber

		// Convert enum types to string
		metadata[client.MoovHolderTypeMetadataKey] = string(transfer.Source.BankAccount.HolderType)
		metadata[client.MoovBankAccountTypeMetadataKey] = string(transfer.Source.BankAccount.BankAccountType)
	}

	// Destination bank account info
	if transfer.Destination.BankAccount != nil {
		metadata[client.MoovDestinationBankAccountIDMetadataKey] = transfer.Destination.BankAccount.BankAccountID
		metadata[client.MoovDestinationHolderNameMetadataKey] = transfer.Destination.BankAccount.HolderName
	}

	// Source ACH details
	if transfer.Source.AchDetails != nil {
		metadata[client.MoovSourceACHStatusMetadataKey] = string(transfer.Source.AchDetails.Status)
		metadata[client.MoovSourceACHTraceNumberMetadataKey] = transfer.Source.AchDetails.TraceNumber
		metadata[client.MoovSourceACHSecCodeMetadataKey] = string(transfer.Source.AchDetails.SecCode)
		metadata[client.MoovSourceACHDebitHoldPeriodMetadataKey] = string(transfer.Source.AchDetails.DebitHoldPeriod)

		// Handle the timestamp formatting safely
		if !transfer.Source.AchDetails.InitiatedOn.IsZero() {
			metadata[client.MoovSourceACHInitiatedOnMetadataKey] = transfer.Source.AchDetails.InitiatedOn.String()
		}
	}

	// Facilitator fee
	if transfer.FacilitatorFee != nil {
		// Convert int64 to float format
		metadata[client.MoovFacilitatorFeeTotalMetadataKey] = fmt.Sprintf("%d", transfer.FacilitatorFee.Total)
		metadata[client.MoovFacilitatorFeeTotalDecimalMetadataKey] = transfer.FacilitatorFee.TotalDecimal
	}

	// Moov fee
	if transfer.MoovFee != nil {
		metadata[client.MoovFeeAmountMetadataKey] = fmt.Sprintf("%d", *transfer.MoovFee)
	}
	if transfer.MoovFeeDecimal != "" {
		metadata[client.MoovFeeAmountDecimalMetadataKey] = transfer.MoovFeeDecimal
	}

	return metadata
}

/*
*
Extracts the facilitator fee from the payment initiation metadata.
*/
func extractFacilitatorFee(pi models.PSPPaymentInitiation) (moov.CreateTransfer_FacilitatorFee, error) {
	facilitatorFeeTotal := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeTotalMetadataKey)
	facilitatorFeeDecimal := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeTotalDecimalMetadataKey)
	facilitatorFeeMarkup := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeMarkupMetadataKey)
	facilitatorFeeMarkupDecimal := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovFacilitatorFeeMarkupDecimalMetadataKey)

	var facilitatorTotal int64
	var err error
	if facilitatorFeeTotal != "" {
		facilitatorTotal, err = strconv.ParseInt(facilitatorFeeTotal, 10, 64)
		if err != nil {
			return moov.CreateTransfer_FacilitatorFee{}, models.NewConnectorValidationError(client.MoovFacilitatorFeeTotalMetadataKey, ErrInvalidFacilitatorFeeTotal)
		}
	}

	var facilitatorTotalDecimal string
	if facilitatorFeeDecimal != "" {
		facilitatorTotalDecimal = facilitatorFeeDecimal
	}

	var facilitatorMarkup int64
	if facilitatorFeeMarkup != "" {
		facilitatorMarkup, err = strconv.ParseInt(facilitatorFeeMarkup, 10, 64)
		if err != nil {
			return moov.CreateTransfer_FacilitatorFee{}, models.NewConnectorValidationError(client.MoovFacilitatorFeeMarkupMetadataKey, ErrInvalidFacilitatorFeeMarkup)
		}
	}

	var facilitatorMarkupDecimal string
	if facilitatorFeeMarkupDecimal != "" {
		facilitatorMarkupDecimal = facilitatorFeeMarkupDecimal
		// Also try to parse the decimal value as an integer for backward compatibility
		if facilitatorFeeMarkup == "" {
			facilitatorMarkup, err = strconv.ParseInt(facilitatorFeeMarkupDecimal, 10, 64)
			if err != nil {
				return moov.CreateTransfer_FacilitatorFee{}, models.NewConnectorValidationError(client.MoovFacilitatorFeeMarkupDecimalMetadataKey, ErrInvalidFacilitatorFeeMarkup)
			}
		}
	}

	facilitatorFee := moov.CreateTransfer_FacilitatorFee{}
	if facilitatorTotal != 0 {
		facilitatorFee.Total = &facilitatorTotal
	}

	if facilitatorTotalDecimal != "" {
		facilitatorFee.TotalDecimal = &facilitatorTotalDecimal
	}

	if facilitatorMarkup != 0 {
		facilitatorFee.Markup = &facilitatorMarkup
	}

	if facilitatorMarkupDecimal != "" {
		facilitatorFee.MarkupDecimal = &facilitatorMarkupDecimal
	}

	return facilitatorFee, nil
}

/*
*
Extracts the sales tax amount from the payment initiation metadata.
*/
func extractSalesTax(pi models.PSPPaymentInitiation) (*moov.Amount, error) {
	salesTaxAmount := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSalesTaxAmountValueMetadataKey)
	salesTaxAmountCurrency := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSalesTaxAmountCurrencyMetadataKey)

	if salesTaxAmount == "" || salesTaxAmountCurrency == "" {
		return nil, nil
	}

	var salesTaxAmountInt int64
	var err error
	if salesTaxAmount != "" {
		salesTaxAmountInt, err = strconv.ParseInt(salesTaxAmount, 10, 64)
		if err != nil {
			return &moov.Amount{}, models.NewConnectorValidationError(client.MoovSalesTaxAmountValueMetadataKey, ErrInvalidSalesTaxAmount)
		}
	}

	curr, _, err := currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, salesTaxAmountCurrency)
	if err != nil {
		return &moov.Amount{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get currency and precision from io.moov.spec/salesTaxAmountCurrency: %w", err),
			models.ErrInvalidRequest,
		)
	}

	return &moov.Amount{
		Currency: curr,
		Value:    salesTaxAmountInt,
	}, nil
}

/*
*

	Extracts the source of the payment from the payment initiation metadata.
*/
func extractPaymentSource(pi models.PSPPaymentInitiation) moov.CreateTransfer_Source {
	secCode := moov.SecCode(models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourceACHSecCodeMetadataKey))
	companyDescription := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourceACHCompanyEntryDescriptionMetadataKey)
	sourceDetails := moov.CreateTransfer_Source{
		PaymentMethodID: models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourcePaymentMethodIDMetadataKey),
		TransferID:      models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourceTransferIDMetadataKey),
	}

	if secCode != "" || companyDescription != "" {
		sourceDetails.AchDetails = &moov.CreateTransfer_AchDetailsSource{
			CompanyEntryDescription: models.ExtractNamespacedMetadata(pi.Metadata, client.MoovSourceACHCompanyEntryDescriptionMetadataKey),
			SecCode:                 &secCode,
		}
	}

	return sourceDetails
}

/*
*
Extracts the destination of the payment from the payment initiation metadata.
*/
func extractPaymentDestination(pi models.PSPPaymentInitiation) moov.CreateTransfer_Destination {
	destinationDetails := moov.CreateTransfer_Destination{
		PaymentMethodID: models.ExtractNamespacedMetadata(pi.Metadata, client.MoovDestinationPaymentMethodIDMetadataKey),
	}

	originatingCompanyName := models.ExtractNamespacedMetadata(pi.Metadata, client.MoovDestinationACHOriginatingCompanyNameMetadataKey)
	if originatingCompanyName != "" {
		destinationDetails.AchDetails = &moov.CreateTransfer_AchDetailsBase{
			CompanyEntryDescription: models.ExtractNamespacedMetadata(pi.Metadata, client.MoovDestinationACHCompanyEntryDescriptionMetadataKey),
			OriginatingCompanyName:  originatingCompanyName,
		}
	}

	return destinationDetails
}

func extractSourceAccountReference(source moov.TransferSource) *string {

	if source.BankAccount != nil {
		return &source.BankAccount.BankAccountID
	}

	if source.Wallet != nil {
		return &source.Wallet.WalletID
	}

	if source.ApplePay != nil {
		return &source.ApplePay.Fingerprint
	}

	if source.Card != nil {
		return &source.Card.CardID
	}

	return nil
}

func extractDestinationAccountReference(destination moov.TransferDestination) *string {

	if destination.BankAccount != nil {
		return &destination.BankAccount.BankAccountID
	}

	if destination.Wallet != nil {
		return &destination.Wallet.WalletID
	}

	if destination.ApplePay != nil {
		return &destination.ApplePay.Fingerprint
	}

	if destination.Card != nil {
		return &destination.Card.CardID
	}

	return nil
}
