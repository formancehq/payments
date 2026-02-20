package connector

import (
	"github.com/formancehq/payments/internal/models"
)

// Core PSP types - these are the data structures connectors produce and consume.
type (
	// PSPAccount represents an account as returned by a Payment Service Provider.
	PSPAccount = models.PSPAccount

	// PSPPayment represents a payment as returned by a Payment Service Provider.
	PSPPayment = models.PSPPayment

	// PSPPaymentsToDelete represents a payment that should be deleted.
	PSPPaymentsToDelete = models.PSPPaymentsToDelete

	// PSPPaymentsToCancel represents a payment that should be cancelled.
	PSPPaymentsToCancel = models.PSPPaymentsToCancel

	// PSPBalance represents a balance as returned by a Payment Service Provider.
	PSPBalance = models.PSPBalance

	// PSPOther represents other data returned by a connector.
	PSPOther = models.PSPOther

	// PSPPaymentInitiation represents a payment initiation request.
	PSPPaymentInitiation = models.PSPPaymentInitiation

	// PSPPaymentInitiationReversal represents a payment initiation reversal.
	PSPPaymentInitiationReversal = models.PSPPaymentInitiationReversal

	// PSPPaymentServiceUser represents a payment service user.
	PSPPaymentServiceUser = models.PSPPaymentServiceUser

	// PSPWebhookConfig represents a webhook configuration from a PSP.
	PSPWebhookConfig = models.PSPWebhookConfig

	// PSPWebhook represents a webhook payload from a PSP.
	PSPWebhook = models.PSPWebhook

	// PSPOpenBankingAccount represents an open banking account.
	PSPOpenBankingAccount = models.PSPOpenBankingAccount

	// PSPOpenBankingPayment represents an open banking payment.
	PSPOpenBankingPayment = models.PSPOpenBankingPayment

	// PSPOpenBankingConnection represents an open banking connection from PSP.
	PSPOpenBankingConnection = models.PSPOpenBankingConnection
)

// Supporting types used by PSP types.
type (
	// Address represents a physical address.
	Address = models.Address

	// ContactDetails represents contact information.
	ContactDetails = models.ContactDetails

	// Token represents an authentication token.
	Token = models.Token

	// BasicAuth represents basic authentication credentials.
	BasicAuth = models.BasicAuth
)

// Metadata helpers - re-exported from internal/models.
var ExtractNamespacedMetadata = models.ExtractNamespacedMetadata

// Bank account metadata keys - re-exported from internal/models.
const (
	BankAccountOwnerAddressLine1MetadataKey = models.BankAccountOwnerAddressLine1MetadataKey
	BankAccountOwnerAddressLine2MetadataKey = models.BankAccountOwnerAddressLine2MetadataKey
	BankAccountOwnerStreetNameMetadataKey   = models.BankAccountOwnerStreetNameMetadataKey
	BankAccountOwnerStreetNumberMetadataKey = models.BankAccountOwnerStreetNumberMetadataKey
	BankAccountOwnerCityMetadataKey         = models.BankAccountOwnerCityMetadataKey
	BankAccountOwnerRegionMetadataKey       = models.BankAccountOwnerRegionMetadataKey
	BankAccountOwnerPostalCodeMetadataKey   = models.BankAccountOwnerPostalCodeMetadataKey
	BankAccountOwnerEmailMetadataKey        = models.BankAccountOwnerEmailMetadataKey
	BankAccountOwnerPhoneNumberMetadataKey  = models.BankAccountOwnerPhoneNumberMetadataKey

	AccountIBANMetadataKey               = models.AccountIBANMetadataKey
	AccountAccountNumberMetadataKey      = models.AccountAccountNumberMetadataKey
	AccountBankAccountNameMetadataKey    = models.AccountBankAccountNameMetadataKey
	AccountBankAccountCountryMetadataKey = models.AccountBankAccountCountryMetadataKey
	AccountSwiftBicCodeMetadataKey       = models.AccountSwiftBicCodeMetadataKey
)

// Open banking event types.
type (
	// PSPDataReadyToFetch indicates that data is ready to be fetched.
	PSPDataReadyToFetch = models.PSPDataReadyToFetch

	// PSPDataToDelete indicates data that should be deleted.
	PSPDataToDelete = models.PSPDataToDelete

	// PSPUserDisconnected indicates a user has been disconnected.
	PSPUserDisconnected = models.PSPUserDisconnected

	// PSPUserConnectionPendingDisconnect indicates a connection is pending disconnection.
	PSPUserConnectionPendingDisconnect = models.PSPUserConnectionPendingDisconnect

	// PSPUserConnectionDisconnected indicates a user connection has been disconnected.
	PSPUserConnectionDisconnected = models.PSPUserConnectionDisconnected

	// PSPUserConnectionReconnected indicates a user connection has been reconnected.
	PSPUserConnectionReconnected = models.PSPUserConnectionReconnected

	// PSPUserLinkSessionFinished indicates a user link session has finished.
	PSPUserLinkSessionFinished = models.PSPUserLinkSessionFinished
)
