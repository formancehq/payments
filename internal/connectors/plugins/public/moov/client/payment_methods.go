package client

type PaymentMethodType string

// Source payment method types
const (
	PaymentMethodTypeACHDebitFund    PaymentMethodType = "ach-debit-fund"
	PaymentMethodTypeACHDebitCollect PaymentMethodType = "ach-debit-collect"
	PaymentMethodTypeApplePay        PaymentMethodType = "apple-pay"
	PaymentMethodTypeMoovWallet      PaymentMethodType = "moov-wallet"
)

// Destination payment method types
const (
	PaymentMethodTypeRTPCredit         PaymentMethodType = "rtp-credit"
	PaymentMethodTypeACHCreditStandard PaymentMethodType = "ach-credit-standard"
	PaymentMethodTypeACHCreditSameDay  PaymentMethodType = "ach-credit-same-day"
)

// FundingSourceType represents the possible funding source types
type FundingSourceType string

const (
	FundingSourceTypeWallet      FundingSourceType = "Wallet"
	FundingSourceTypeBankAccount FundingSourceType = "Bank account"
)

// PaymentMethodInfo contains details about a payment method
type PaymentMethodInfo struct {
	Limit float64
}

// PaymentMethodMap maps payment method types to their details
var PaymentMethodMap = map[PaymentMethodType]PaymentMethodInfo{
	// Source payment methods

	/**
	Fund payouts or add funds from a linked bank account
	*/
	PaymentMethodTypeACHDebitFund: {
		Limit: 99_999_999.99, // Standard: $99,999,999.99, Same day: $1,000,000
	},

	/**
	Pull funds for bill payment, direct debit, or e-check type use-cases
	*/
	PaymentMethodTypeACHDebitCollect: {
		Limit: 99_999_999.99, // Standard: $99,999,999.99, Same day: $1,000,000
	},

	/**
	Facilitate an Apple Pay transaction to a Moov account
	*/
	PaymentMethodTypeApplePay: {
		Limit: 99_999_999.99, // $99,999,999.99
	},

	/**
	Fund payouts or withdraw funds from the Moov platform
	*/
	PaymentMethodTypeMoovWallet: {
		Limit: 99_999_999.99, // Lesser of $99,999,999.99 or wallet availableBalance
	},

	// Destination payment methods

	/**
	Disburse funds to a linked bank account in near real time
	*/
	PaymentMethodTypeRTPCredit: {
		Limit: 99_999_999.99, // $99,999,999.99
	},

	/**
	Disburse funds to a linked bank account
	*/
	PaymentMethodTypeACHCreditStandard: {
		Limit: 99_999_999.99, // $99,999,999.99
	},

	/**
	Disburse funds to a linked bank account using same-day processing
	*/
	PaymentMethodTypeACHCreditSameDay: {
		Limit: 500_000, // $500,000
	},
}

// ValidateTransactionLimit checks if a transaction amount is within the limit for a payment method type
func ValidateTransactionLimit(pmType PaymentMethodType, amount float64) (bool, float64) {
	if info, exists := PaymentMethodMap[pmType]; exists {
		return amount <= info.Limit, info.Limit
	}
	return false, 0
}
