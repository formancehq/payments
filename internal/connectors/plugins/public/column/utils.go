package column

import (
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) validateTransferRequest(pi models.PSPPaymentInitiation) error {

	if pi.Amount == nil {
		return models.NewConnectorValidationError("amount", ErrMissingAmount)
	}

	if pi.Asset == "" {
		return models.NewConnectorValidationError("asset", ErrMissingAsset)
	}

	if pi.Metadata == nil {
		return models.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	if pi.SourceAccount == nil {
		return models.NewConnectorValidationError("sourceAccount", ErrMissingSourceAccount)
	}

	if pi.SourceAccount.Name == nil {
		return models.NewConnectorValidationError("sourceAccount.name", ErrMissingSourceAccountName)
	}

	if pi.SourceAccount.Reference == "" {
		return models.NewConnectorValidationError("sourceAccount.reference", ErrSourceAccountReferenceRequired)
	}

	if pi.DestinationAccount == nil {
		return models.NewConnectorValidationError("destinationAccount", ErrMissingDestinationAccount)
	}

	if pi.DestinationAccount.Reference == "" {
		return models.NewConnectorValidationError("destinationAccount.reference", ErrMissingDestinationAccountReference)
	}

	allowOverdraft := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAllowOverdraftMetadataKey)
	if allowOverdraft != "" {
		if allowOverdraft != "true" && allowOverdraft != "false" {
			return models.NewConnectorValidationError(client.ColumnAllowOverdraftMetadataKey, ErrMissingMetadataAllowOverDrafts)
		}
	}

	hold := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnHoldMetadataKey)
	if hold != "" {
		if hold != "true" && hold != "false" {
			return models.NewConnectorValidationError(client.ColumnHoldMetadataKey, ErrMissingMetadataHold)

		}
	}

	countryCode := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAddressCountryCodeMetadataKey)
	err := validateAddress(pi.Metadata, countryCode, true)

	if err != nil {
		return err
	}

	return nil
}

func validateAddress(addressMetadata map[string]string, country string, isCountryInMetadata bool) error {
	addressLine1 := models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressLine1MetadataKey)

	if addressLine1 != "" {
		city := models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressCityMetadataKey)

		if city == "" {
			return models.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMissingMetadataAddressCity)
		}

		if country == "" {
			if isCountryInMetadata {
				return models.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMissingMetadataCountry)
			} else {
				return models.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMissingCountry)
			}
		}
		return nil
	}

	if models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressLine2MetadataKey) != "" {
		return models.NewConnectorValidationError(client.ColumnAddressLine2MetadataKey, ErrMetadataAddressLine2NotRequired)
	}

	if models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressCityMetadataKey) != "" {
		return models.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMetadataAddressCityNotRequired)
	}

	if models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressStateMetadataKey) != "" {
		return models.NewConnectorValidationError(client.ColumnAddressStateMetadataKey, ErrMetadataAddressStateNotRequired)
	}

	if models.ExtractNamespacedMetadata(addressMetadata, client.ColumnAddressPostalCodeMetadataKey) != "" {
		return models.NewConnectorValidationError(client.ColumnAddressPostalCodeMetadataKey, ErrMetadataPostalCodeNotRequired)
	}

	if country != "" {
		if isCountryInMetadata {
			return models.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMetadataAddressCountryNotRequired)
		} else {
			return models.NewConnectorValidationError("country", ErrCountryNotRequired)
		}
	}

	return nil
}

func (p *Plugin) validatePayoutRequests(pi models.PSPPaymentInitiation) error {

	if pi.Amount == nil {
		return models.NewConnectorValidationError("Amount", ErrMissingAmount)
	}

	if pi.SourceAccount == nil {
		return models.NewConnectorValidationError("SourceAccount", ErrMissingSourceAccount)
	}

	if pi.SourceAccount.Reference == "" {
		return models.NewConnectorValidationError("SourceAccount.Reference", ErrSourceAccountReferenceRequired)
	}

	if pi.DestinationAccount == nil {
		return models.NewConnectorValidationError("DestinationAccount", ErrMissingDestinationAccount)
	}

	if pi.DestinationAccount.Reference == "" {
		return models.NewConnectorValidationError("DestinationAccount.Reference", ErrMissingDestinationAccountReference)
	}

	if pi.Metadata == nil {
		return models.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	payoutType := models.ExtractNamespacedMetadata(pi.Metadata, client.ColumnPayoutTypeMetadataKey)

	if payoutType == "" {
		return models.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	if payoutType != "ach" && payoutType != "wire" && payoutType != "realtime" && payoutType != "international-wire" {
		return models.NewConnectorValidationError(client.ColumnPayoutTypeMetadataKey, ErrInvalidMetadataPayoutType)
	}

	if pi.Asset == "" {
		return models.NewConnectorValidationError("asset", ErrMissingAsset)
	}

	return nil
}

func (p *Plugin) validateReversePayout(pr models.PSPPaymentInitiationReversal) error {
	if pr.Metadata == nil {
		return models.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	reason := models.ExtractNamespacedMetadata(pr.Metadata, client.ColumnReasonMetadataKey)
	if reason == "" {
		return models.NewConnectorValidationError(client.ColumnReasonMetadataKey, ErrMissingMetadataReason)
	}

	if !IsValidReversePayoutReason(reason) {
		return models.NewConnectorValidationError(client.ColumnReasonMetadataKey, ErrInvalidMetadataReason)
	}

	if pr.RelatedPaymentInitiation.Reference == "" {
		return models.NewConnectorValidationError("relatedPaymentInitiation.reference", ErrMissingRelatedPaymentInitiationReference)
	}

	return nil
}

func (p *Plugin) validateExternalBankAccount(newExternalBankAccount models.BankAccount) error {
	if newExternalBankAccount.AccountNumber == nil {
		return models.NewConnectorValidationError("AccountNumber", ErrAccountNumberRequired)
	}

	routingNumber := models.ExtractNamespacedMetadata(newExternalBankAccount.Metadata, client.ColumnRoutingNumberMetadataKey)

	if routingNumber == "" {
		return models.NewConnectorValidationError(client.ColumnRoutingNumberMetadataKey, ErrMissingRoutingNumber)
	}

	country := ""
	if newExternalBankAccount.Country != nil {
		country = *newExternalBankAccount.Country
	}

	err := validateAddress(newExternalBankAccount.Metadata, country, false)

	if err != nil {
		return err
	}

	return nil
}

func ParseColumnTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

func (p *Plugin) matchStatus(status string) models.PaymentStatus {
	switch status {
	case "REJECTED":
		return models.PAYMENT_STATUS_FAILED
	case "COMPLETED":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "HOLD":
		return models.PAYMENT_STATUS_PENDING
	case "CANCELED":
		return models.PAYMENT_STATUS_CANCELLED
	}

	return models.PAYMENT_STATUS_UNKNOWN
}

func IsValidReversePayoutReason(reason string) bool {
	switch client.ReversePayoutReason(reason) {
	case client.ReasonDuplicatedEntry,
		client.ReasonIncorrectAmount,
		client.ReasonIncorrectReceiverAccount,
		client.ReasonDebitEarlierThanIntended,
		client.ReasonCreditLaterThanIntended:
		return true
	}
	return false
}
