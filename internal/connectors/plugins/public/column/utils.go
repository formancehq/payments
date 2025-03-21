package column

import (
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/column/client"
	"github.com/formancehq/payments/internal/models"
)

type ColumnAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

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
	address := extractAddressFromMetadata(pi.Metadata, countryCode)

	return validateAddressForTransfer(address)
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

	address := extractAddressFromMetadata(newExternalBankAccount.Metadata, country)
	return validateAddressForBankAccount(address)
}

func validateAddressForTransfer(address ColumnAddress) error {
	// If Line1 is provided, we need a complete address
	if address.Line1 != "" {
		if address.City == "" {
			return models.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMissingMetadataAddressCity)
		}

		if address.Country == "" {
			return models.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMissingMetadataCountry)
		}

		return nil
	}

	// No address line provided, ensure no other address fields are provided
	if address.Line2 != "" {
		return models.NewConnectorValidationError(client.ColumnAddressLine2MetadataKey, ErrMetadataAddressLine2NotRequired)
	}

	if address.City != "" {
		return models.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMetadataAddressCityNotRequired)
	}

	if address.State != "" {
		return models.NewConnectorValidationError(client.ColumnAddressStateMetadataKey, ErrMetadataAddressStateNotRequired)
	}

	if address.PostalCode != "" {
		return models.NewConnectorValidationError(client.ColumnAddressPostalCodeMetadataKey, ErrMetadataPostalCodeNotRequired)
	}

	if address.Country != "" {
		return models.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMetadataAddressCountryNotRequired)
	}

	return nil
}

func validateAddressForBankAccount(address ColumnAddress) error {
	// If Line1 is provided, we need a complete address
	if address.Line1 != "" {
		if address.City == "" {
			return models.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMissingMetadataAddressCity)
		}

		if address.Country == "" {
			return models.NewConnectorValidationError("country", ErrMissingCountry)
		}

		return nil
	}

	// No address line provided, ensure no other address fields are provided
	if address.Line2 != "" {
		return models.NewConnectorValidationError(client.ColumnAddressLine2MetadataKey, ErrMetadataAddressLine2NotRequired)
	}

	if address.City != "" {
		return models.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMetadataAddressCityNotRequired)
	}

	if address.State != "" {
		return models.NewConnectorValidationError(client.ColumnAddressStateMetadataKey, ErrMetadataAddressStateNotRequired)
	}

	if address.PostalCode != "" {
		return models.NewConnectorValidationError(client.ColumnAddressPostalCodeMetadataKey, ErrMetadataPostalCodeNotRequired)
	}

	if address.Country != "" {
		return models.NewConnectorValidationError("country", ErrCountryNotRequired)
	}

	return nil
}

func extractAddressFromMetadata(metadata map[string]string, country string) ColumnAddress {
	return ColumnAddress{
		Line1:      models.ExtractNamespacedMetadata(metadata, client.ColumnAddressLine1MetadataKey),
		Line2:      models.ExtractNamespacedMetadata(metadata, client.ColumnAddressLine2MetadataKey),
		City:       models.ExtractNamespacedMetadata(metadata, client.ColumnAddressCityMetadataKey),
		State:      models.ExtractNamespacedMetadata(metadata, client.ColumnAddressStateMetadataKey),
		PostalCode: models.ExtractNamespacedMetadata(metadata, client.ColumnAddressPostalCodeMetadataKey),
		Country:    country,
	}
}

func ParseColumnTimestamp(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
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
