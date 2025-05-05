package client

const (
	GocardlessMetadataSpecNamespace = "com.gocardless.spec/"

	GocardlessCurrencyMetadataKey    = GocardlessMetadataSpecNamespace + "currency"
	GocardlessCreditorMetadataKey    = GocardlessMetadataSpecNamespace + "creditor"
	GocardlessCustomerMetadataKey    = GocardlessMetadataSpecNamespace + "customer"
	GocardlessAccountTypeMetadataKey = GocardlessMetadataSpecNamespace + "account_type"

	GocardlessFxEstimatedExchangeRateMetadataKey = GocardlessMetadataSpecNamespace + "fx_estimated_exchange_rate"
	GocardlessFxExchangeRateMetadataKey          = GocardlessMetadataSpecNamespace + "fx_exchange_rate"
	GoCardlessFxAmountMetadataKey                = GocardlessMetadataSpecNamespace + "fx_amount"
	GoCardlessFxCurrencyMetadataKey              = GocardlessMetadataSpecNamespace + "fx_currency"

	GocardlessLinksMetadataKey                  = GocardlessMetadataSpecNamespace + "links"
	GocardlessLinkCreditorMetadataKey           = GocardlessMetadataSpecNamespace + "links_creditor"
	GocardlessLinkInstalmentScheduleMetadataKey = GocardlessMetadataSpecNamespace + "links_instalment_schedule"
	GocardlessMandateMetadataKey                = GocardlessMetadataSpecNamespace + "links_mandate"
	GocardlessPayoutMetadataKey                 = GocardlessMetadataSpecNamespace + "links_payout"
	GocardlessSubscriptionMetadataKey           = GocardlessMetadataSpecNamespace + "links_subscription"

	GocardlessAmountRefundedMetadataKey  = GocardlessMetadataSpecNamespace + "amount_refunded"
	GocardlessChargeDateMetadataKey      = GocardlessMetadataSpecNamespace + "charge_date"
	GocardlessDescriptionMetadataKey     = GocardlessMetadataSpecNamespace + "description"
	GocardlessFasterAchMetadataKey       = GocardlessMetadataSpecNamespace + "faster_ach"
	GocardlessRetryIfPossibleMetadataKey = GocardlessMetadataSpecNamespace + "retry_if_possible"
	GocardlessReferenceMetadataKey       = GocardlessMetadataSpecNamespace + "reference"
)
