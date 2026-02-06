package connector

import (
	"github.com/formancehq/payments/internal/models"
)

// Payment type enum.
type PaymentType = models.PaymentType

const (
	PAYMENT_TYPE_UNKNOWN  = models.PAYMENT_TYPE_UNKNOWN
	PAYMENT_TYPE_PAYIN    = models.PAYMENT_TYPE_PAYIN
	PAYMENT_TYPE_PAYOUT   = models.PAYMENT_TYPE_PAYOUT
	PAYMENT_TYPE_TRANSFER = models.PAYMENT_TYPE_TRANSFER
	PAYMENT_TYPE_OTHER    = models.PAYMENT_TYPE_OTHER
)

var (
	PaymentTypeFromString     = models.PaymentTypeFromString
	MustPaymentTypeFromString = models.MustPaymentTypeFromString
)

// Payment status enum.
type PaymentStatus = models.PaymentStatus

const (
	PAYMENT_STATUS_UNKNOWN           = models.PAYMENT_STATUS_UNKNOWN
	PAYMENT_STATUS_PENDING           = models.PAYMENT_STATUS_PENDING
	PAYMENT_STATUS_SUCCEEDED         = models.PAYMENT_STATUS_SUCCEEDED
	PAYMENT_STATUS_CANCELLED         = models.PAYMENT_STATUS_CANCELLED
	PAYMENT_STATUS_FAILED            = models.PAYMENT_STATUS_FAILED
	PAYMENT_STATUS_EXPIRED           = models.PAYMENT_STATUS_EXPIRED
	PAYMENT_STATUS_REFUNDED          = models.PAYMENT_STATUS_REFUNDED
	PAYMENT_STATUS_REFUNDED_FAILURE  = models.PAYMENT_STATUS_REFUNDED_FAILURE
	PAYMENT_STATUS_REFUND_REVERSED   = models.PAYMENT_STATUS_REFUND_REVERSED
	PAYMENT_STATUS_DISPUTE           = models.PAYMENT_STATUS_DISPUTE
	PAYMENT_STATUS_DISPUTE_WON       = models.PAYMENT_STATUS_DISPUTE_WON
	PAYMENT_STATUS_DISPUTE_LOST      = models.PAYMENT_STATUS_DISPUTE_LOST
	PAYMENT_STATUS_AMOUNT_ADJUSTMENT = models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT
	PAYMENT_STATUS_AUTHORISATION     = models.PAYMENT_STATUS_AUTHORISATION
	PAYMENT_STATUS_CAPTURE           = models.PAYMENT_STATUS_CAPTURE
	PAYMENT_STATUS_CAPTURE_FAILED    = models.PAYMENT_STATUS_CAPTURE_FAILED
	PAYMENT_STATUS_OTHER             = models.PAYMENT_STATUS_OTHER
)

var (
	PaymentStatusFromString     = models.PaymentStatusFromString
	MustPaymentStatusFromString = models.MustPaymentStatusFromString
)

// Payment scheme enum.
type PaymentScheme = models.PaymentScheme

const (
	PAYMENT_SCHEME_UNKNOWN         = models.PAYMENT_SCHEME_UNKNOWN
	PAYMENT_SCHEME_CARD_VISA       = models.PAYMENT_SCHEME_CARD_VISA
	PAYMENT_SCHEME_CARD_MASTERCARD = models.PAYMENT_SCHEME_CARD_MASTERCARD
	PAYMENT_SCHEME_CARD_AMEX       = models.PAYMENT_SCHEME_CARD_AMEX
	PAYMENT_SCHEME_CARD_DINERS     = models.PAYMENT_SCHEME_CARD_DINERS
	PAYMENT_SCHEME_CARD_DISCOVER   = models.PAYMENT_SCHEME_CARD_DISCOVER
	PAYMENT_SCHEME_CARD_JCB        = models.PAYMENT_SCHEME_CARD_JCB
	PAYMENT_SCHEME_CARD_UNION_PAY  = models.PAYMENT_SCHEME_CARD_UNION_PAY
	PAYMENT_SCHEME_CARD_ALIPAY     = models.PAYMENT_SCHEME_CARD_ALIPAY
	PAYMENT_SCHEME_CARD_CUP        = models.PAYMENT_SCHEME_CARD_CUP
	PAYMENT_SCHEME_SEPA_DEBIT      = models.PAYMENT_SCHEME_SEPA_DEBIT
	PAYMENT_SCHEME_SEPA_CREDIT     = models.PAYMENT_SCHEME_SEPA_CREDIT
	PAYMENT_SCHEME_SEPA            = models.PAYMENT_SCHEME_SEPA
	PAYMENT_SCHEME_GOOGLE_PAY      = models.PAYMENT_SCHEME_GOOGLE_PAY
	PAYMENT_SCHEME_APPLE_PAY       = models.PAYMENT_SCHEME_APPLE_PAY
	PAYMENT_SCHEME_DOKU            = models.PAYMENT_SCHEME_DOKU
	PAYMENT_SCHEME_DRAGON_PAY      = models.PAYMENT_SCHEME_DRAGON_PAY
	PAYMENT_SCHEME_MAESTRO         = models.PAYMENT_SCHEME_MAESTRO
	PAYMENT_SCHEME_MOL_PAY         = models.PAYMENT_SCHEME_MOL_PAY
	PAYMENT_SCHEME_A2A             = models.PAYMENT_SCHEME_A2A
	PAYMENT_SCHEME_ACH_DEBIT       = models.PAYMENT_SCHEME_ACH_DEBIT
	PAYMENT_SCHEME_ACH             = models.PAYMENT_SCHEME_ACH
	PAYMENT_SCHEME_RTP             = models.PAYMENT_SCHEME_RTP
	PAYMENT_SCHEME_OTHER           = models.PAYMENT_SCHEME_OTHER
)

var (
	PaymentSchemeFromString     = models.PaymentSchemeFromString
	MustPaymentSchemeFromString = models.MustPaymentSchemeFromString
)

// Capability enum.
type Capability = models.Capability

const (
	CAPABILITY_FETCH_UNKNOWN                   = models.CAPABILITY_FETCH_UNKNOWN
	CAPABILITY_FETCH_ACCOUNTS                  = models.CAPABILITY_FETCH_ACCOUNTS
	CAPABILITY_FETCH_BALANCES                  = models.CAPABILITY_FETCH_BALANCES
	CAPABILITY_FETCH_EXTERNAL_ACCOUNTS         = models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS
	CAPABILITY_FETCH_PAYMENTS                  = models.CAPABILITY_FETCH_PAYMENTS
	CAPABILITY_FETCH_OTHERS                    = models.CAPABILITY_FETCH_OTHERS
	CAPABILITY_CREATE_WEBHOOKS                 = models.CAPABILITY_CREATE_WEBHOOKS
	CAPABILITY_TRANSLATE_WEBHOOKS              = models.CAPABILITY_TRANSLATE_WEBHOOKS
	CAPABILITY_CREATE_BANK_ACCOUNT             = models.CAPABILITY_CREATE_BANK_ACCOUNT
	CAPABILITY_CREATE_TRANSFER                 = models.CAPABILITY_CREATE_TRANSFER
	CAPABILITY_CREATE_PAYOUT                   = models.CAPABILITY_CREATE_PAYOUT
	CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION = models.CAPABILITY_ALLOW_FORMANCE_ACCOUNT_CREATION
	CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION = models.CAPABILITY_ALLOW_FORMANCE_PAYMENT_CREATION
)

// Open banking enums.
type (
	OpenBankingDataToFetch             = models.OpenBankingDataToFetch
	ConnectionDisconnectedErrorType    = models.ConnectionDisconnectedErrorType
	OpenBankingConnectionAttemptStatus = models.OpenBankingConnectionAttemptStatus
	ConnectionStatus                   = models.ConnectionStatus
)

const (
	OpenBankingDataToFetchPayments            = models.OpenBankingDataToFetchPayments
	OpenBankingDataToFetchAccountsAndBalances = models.OpenBankingDataToFetchAccountsAndBalances

	ConnectionDisconnectedErrorTypeTemporaryError   = models.ConnectionDisconnectedErrorTypeTemporaryError
	ConnectionDisconnectedErrorTypeNonRecoverable   = models.ConnectionDisconnectedErrorTypeNonRecoverable
	ConnectionDisconnectedErrorTypeUserActionNeeded = models.ConnectionDisconnectedErrorTypeUserActionNeeded

	OpenBankingConnectionAttemptStatusPending   = models.OpenBankingConnectionAttemptStatusPending
	OpenBankingConnectionAttemptStatusCompleted = models.OpenBankingConnectionAttemptStatusCompleted
	OpenBankingConnectionAttemptStatusExited    = models.OpenBankingConnectionAttemptStatusExited

	ConnectionStatusActive = models.ConnectionStatusActive
	ConnectionStatusError  = models.ConnectionStatusError
)
