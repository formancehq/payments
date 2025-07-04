package client

const (
	moovMetadataSpecNamespace = "io.moov.spec/"

	MoovWalletCurrencyMetadataKey                   = moovMetadataSpecNamespace + "walletCurrency"
	MoovWalletValueMetadataKey                      = moovMetadataSpecNamespace + "walletValue"
	MoovValueDecimalMetadataKey                     = moovMetadataSpecNamespace + "valueDecimal"
	MoovAccountIDMetadataKey                        = moovMetadataSpecNamespace + "accountId"
	MoovFingerprintMetadataKey                      = moovMetadataSpecNamespace + "fingerprint"
	MoovStatusMetadataKey                           = moovMetadataSpecNamespace + "status"
	MoovBankNameMetadataKey                         = moovMetadataSpecNamespace + "bankName"
	MoovHolderTypeMetadataKey                       = moovMetadataSpecNamespace + "holderType"
	MoovBankAccountTypeMetadataKey                  = moovMetadataSpecNamespace + "bankAccountType"
	MoovRoutingNumberMetadataKey                    = moovMetadataSpecNamespace + "routingNumber"
	MoovLastFourAccountNumberMetadataKey            = moovMetadataSpecNamespace + "lastFourAccountNumber"
	MoovUpdateOnMetadataKey                         = moovMetadataSpecNamespace + "updatedOn"
	MoovStatusReasonMetadataKey                     = moovMetadataSpecNamespace + "statusReason"
	MoovExceptionDetailsAchReturnCodeMetadataKey    = moovMetadataSpecNamespace + "exceptionDetailsAchReturnCode"
	MoovExceptionDetailsDescriptionMetadataKey      = moovMetadataSpecNamespace + "exceptionDetailsDescription"
	MoovExceptionDetailsRTPRejectionCodeMetadataKey = moovMetadataSpecNamespace + "exceptionDetailsRTPRejectionCode"

	// Payment type
	MoovPaymentTypeMetadataKey = moovMetadataSpecNamespace + "type"

	// Destination payment method metadata keys
	MoovDestinationPaymentMethodIDMetadataKey            = moovMetadataSpecNamespace + "destinationPaymentMethodId"
	MoovDestinationACHCompanyEntryDescriptionMetadataKey = moovMetadataSpecNamespace + "destinationACHCompanyEntryDescription"
	MoovDestinationACHOriginatingCompanyNameMetadataKey  = moovMetadataSpecNamespace + "destinationACHOriginatingCompanyName"
	MoovDestinationPaymentMethodTypeMetadataKey          = moovMetadataSpecNamespace + "destinationPaymentMethodType"
	MoovDestinationAccountEmailMetadataKey               = moovMetadataSpecNamespace + "destinationAccountEmail"
	MoovDestinationAccountDisplayNameMetadataKey         = moovMetadataSpecNamespace + "destinationAccountDisplayName"
	MoovDestinationBankAccountIDMetadataKey              = moovMetadataSpecNamespace + "destinationBankAccountId"
	MoovDestinationHolderNameMetadataKey                 = moovMetadataSpecNamespace + "destinationHolderName"
	MoovDestinationWalletIDMetadataKey                   = moovMetadataSpecNamespace + "destinationWalletId"

	// Source payment method metadata keys
	MoovSourcePaymentMethodIDMetadataKey            = moovMetadataSpecNamespace + "sourcePaymentMethodId"
	MoovSourceACHCompanyEntryDescriptionMetadataKey = moovMetadataSpecNamespace + "sourceACHCompanyEntryDescription"
	MoovSourceACHDebitHoldPeriodMetadataKey         = moovMetadataSpecNamespace + "sourceACHDebitHoldPeriod"
	MoovSourceACHSecCodeMetadataKey                 = moovMetadataSpecNamespace + "sourceACHSecCode"
	MoovSourceTransferIDMetadataKey                 = moovMetadataSpecNamespace + "sourceTransferID"
	MoovSourcePaymentMethodTypeMetadataKey          = moovMetadataSpecNamespace + "sourcePaymentMethodType"
	MoovSourceAccountEmailMetadataKey               = moovMetadataSpecNamespace + "sourceAccountEmail"
	MoovSourceAccountDisplayNameMetadataKey         = moovMetadataSpecNamespace + "sourceAccountDisplayName"
	MoovSourceBankAccountIDMetadataKey              = moovMetadataSpecNamespace + "sourceBankAccountId"
	MoovSourceHolderNameMetadataKey                 = moovMetadataSpecNamespace + "sourceHolderName"
	MoovSourceACHTraceNumberMetadataKey             = moovMetadataSpecNamespace + "sourceACHTraceNumber"
	MoovSourceACHStatusMetadataKey                  = moovMetadataSpecNamespace + "sourceACHStatus"
	MoovSourceACHInitiatedOnMetadataKey             = moovMetadataSpecNamespace + "sourceACHInitiatedOn"

	// Facilitator fee metadata keys
	MoovFacilitatorFeeMarkupMetadataKey        = moovMetadataSpecNamespace + "facilitatorFeeMarkup"
	MoovFacilitatorFeeMarkupDecimalMetadataKey = moovMetadataSpecNamespace + "facilitatorFeeMarkupDecimal"
	MoovFacilitatorFeeTotalMetadataKey         = moovMetadataSpecNamespace + "facilitatorFeeTotal"
	MoovFacilitatorFeeTotalDecimalMetadataKey  = moovMetadataSpecNamespace + "facilitatorFeeTotalDecimal"

	// Sales tax metadata keys
	MoovSalesTaxAmountCurrencyMetadataKey = moovMetadataSpecNamespace + "salesTaxAmountCurrency"
	MoovSalesTaxAmountValueMetadataKey    = moovMetadataSpecNamespace + "salesTaxAmountvalue"

	// Moov fee metadata keys
	MoovFeeAmountMetadataKey        = moovMetadataSpecNamespace + "moovFeeAmount"
	MoovFeeAmountDecimalMetadataKey = moovMetadataSpecNamespace + "moovFeeAmountDecimal"
)
