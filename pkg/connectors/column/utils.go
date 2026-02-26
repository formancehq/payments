package column

import (
	"time"

	"github.com/formancehq/payments/pkg/connectors/column/client"
	"github.com/formancehq/payments/pkg/connector"
)

type ColumnAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

func (p *Plugin) validateTransferRequest(pi connector.PSPPaymentInitiation) error {

	if pi.Amount == nil {
		return connector.NewConnectorValidationError("amount", ErrMissingAmount)
	}

	if pi.Asset == "" {
		return connector.NewConnectorValidationError("asset", ErrMissingAsset)
	}

	if pi.Metadata == nil {
		return connector.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	if pi.SourceAccount == nil {
		return connector.NewConnectorValidationError("sourceAccount", ErrMissingSourceAccount)
	}

	if pi.SourceAccount.Name == nil {
		return connector.NewConnectorValidationError("sourceAccount.name", ErrMissingSourceAccountName)
	}

	if pi.SourceAccount.Reference == "" {
		return connector.NewConnectorValidationError("sourceAccount.reference", ErrSourceAccountReferenceRequired)
	}

	if pi.DestinationAccount == nil {
		return connector.NewConnectorValidationError("destinationAccount", ErrMissingDestinationAccount)
	}

	if pi.DestinationAccount.Reference == "" {
		return connector.NewConnectorValidationError("destinationAccount.reference", ErrMissingDestinationAccountReference)
	}

	allowOverdraft := connector.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAllowOverdraftMetadataKey)
	if allowOverdraft != "" {
		if allowOverdraft != "true" && allowOverdraft != "false" {
			return connector.NewConnectorValidationError(client.ColumnAllowOverdraftMetadataKey, ErrMissingMetadataAllowOverDrafts)
		}
	}

	hold := connector.ExtractNamespacedMetadata(pi.Metadata, client.ColumnHoldMetadataKey)
	if hold != "" {
		if hold != "true" && hold != "false" {
			return connector.NewConnectorValidationError(client.ColumnHoldMetadataKey, ErrMissingMetadataHold)

		}
	}

	countryCode := connector.ExtractNamespacedMetadata(pi.Metadata, client.ColumnAddressCountryCodeMetadataKey)
	address := extractAddressFromMetadata(pi.Metadata, countryCode)

	return validateAddressForTransfer(address)
}

func (p *Plugin) validatePayoutRequests(pi connector.PSPPaymentInitiation) error {

	if pi.Amount == nil {
		return connector.NewConnectorValidationError("Amount", ErrMissingAmount)
	}

	if pi.SourceAccount == nil {
		return connector.NewConnectorValidationError("SourceAccount", ErrMissingSourceAccount)
	}

	if pi.SourceAccount.Reference == "" {
		return connector.NewConnectorValidationError("SourceAccount.Reference", ErrSourceAccountReferenceRequired)
	}

	if pi.DestinationAccount == nil {
		return connector.NewConnectorValidationError("DestinationAccount", ErrMissingDestinationAccount)
	}

	if pi.DestinationAccount.Reference == "" {
		return connector.NewConnectorValidationError("DestinationAccount.Reference", ErrMissingDestinationAccountReference)
	}

	if pi.Metadata == nil {
		return connector.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	payoutType := connector.ExtractNamespacedMetadata(pi.Metadata, client.ColumnPayoutTypeMetadataKey)

	if payoutType == "" {
		return connector.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	if payoutType != "ach" && payoutType != "wire" && payoutType != "realtime" && payoutType != "international-wire" {
		return connector.NewConnectorValidationError(client.ColumnPayoutTypeMetadataKey, ErrInvalidMetadataPayoutType)
	}

	if pi.Asset == "" {
		return connector.NewConnectorValidationError("asset", ErrMissingAsset)
	}

	return nil
}

func (p *Plugin) validateReversePayout(pr connector.PSPPaymentInitiationReversal) error {
	if pr.Metadata == nil {
		return connector.NewConnectorValidationError("metadata", ErrMissingMetadata)
	}

	reason := connector.ExtractNamespacedMetadata(pr.Metadata, client.ColumnReasonMetadataKey)
	if reason == "" {
		return connector.NewConnectorValidationError(client.ColumnReasonMetadataKey, ErrMissingMetadataReason)
	}

	if !IsValidReversePayoutReason(reason) {
		return connector.NewConnectorValidationError(client.ColumnReasonMetadataKey, ErrInvalidMetadataReason)
	}

	if pr.RelatedPaymentInitiation.Reference == "" {
		return connector.NewConnectorValidationError("relatedPaymentInitiation.reference", ErrMissingRelatedPaymentInitiationReference)
	}

	return nil
}

func (p *Plugin) validateExternalBankAccount(newExternalBankAccount connector.BankAccount) error {
	if newExternalBankAccount.AccountNumber == nil {
		return connector.NewConnectorValidationError("AccountNumber", ErrAccountNumberRequired)
	}

	routingNumber := connector.ExtractNamespacedMetadata(newExternalBankAccount.Metadata, client.ColumnRoutingNumberMetadataKey)

	if routingNumber == "" {
		return connector.NewConnectorValidationError(client.ColumnRoutingNumberMetadataKey, ErrMissingRoutingNumber)
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
			return connector.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMissingMetadataAddressCity)
		}

		if address.Country == "" {
			return connector.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMissingMetadataCountry)
		}

		return nil
	}

	// No address line provided, ensure no other address fields are provided
	if address.Line2 != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressLine2MetadataKey, ErrMetadataAddressLine2NotRequired)
	}

	if address.City != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMetadataAddressCityNotRequired)
	}

	if address.State != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressStateMetadataKey, ErrMetadataAddressStateNotRequired)
	}

	if address.PostalCode != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressPostalCodeMetadataKey, ErrMetadataPostalCodeNotRequired)
	}

	if address.Country != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressCountryCodeMetadataKey, ErrMetadataAddressCountryNotRequired)
	}

	return nil
}

func validateAddressForBankAccount(address ColumnAddress) error {
	// If Line1 is provided, we need a complete address
	if address.Line1 != "" {
		if address.City == "" {
			return connector.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMissingMetadataAddressCity)
		}

		if address.Country == "" {
			return connector.NewConnectorValidationError("country", ErrMissingCountry)
		}

		return nil
	}

	// No address line provided, ensure no other address fields are provided
	if address.Line2 != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressLine2MetadataKey, ErrMetadataAddressLine2NotRequired)
	}

	if address.City != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressCityMetadataKey, ErrMetadataAddressCityNotRequired)
	}

	if address.State != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressStateMetadataKey, ErrMetadataAddressStateNotRequired)
	}

	if address.PostalCode != "" {
		return connector.NewConnectorValidationError(client.ColumnAddressPostalCodeMetadataKey, ErrMetadataPostalCodeNotRequired)
	}

	if address.Country != "" {
		return connector.NewConnectorValidationError("country", ErrCountryNotRequired)
	}

	return nil
}

func extractAddressFromMetadata(metadata map[string]string, country string) ColumnAddress {
	return ColumnAddress{
		Line1:      connector.ExtractNamespacedMetadata(metadata, client.ColumnAddressLine1MetadataKey),
		Line2:      connector.ExtractNamespacedMetadata(metadata, client.ColumnAddressLine2MetadataKey),
		City:       connector.ExtractNamespacedMetadata(metadata, client.ColumnAddressCityMetadataKey),
		State:      connector.ExtractNamespacedMetadata(metadata, client.ColumnAddressStateMetadataKey),
		PostalCode: connector.ExtractNamespacedMetadata(metadata, client.ColumnAddressPostalCodeMetadataKey),
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
