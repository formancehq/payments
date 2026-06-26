package bankingbridge_test

import (
	"testing"

	"github.com/formancehq/payments/ee/plugins/bankingbridge"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/assert"
)

func TestPaymentSchemeAndType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		code           string
		expectedScheme models.PaymentScheme
		expectedType   models.PaymentType
	}{
		{
			name:           "Unknown code",
			code:           "UNKNOWN.SCHEME",
			expectedScheme: models.PAYMENT_SCHEME_UNKNOWN,
			expectedType:   models.PAYMENT_TYPE_UNKNOWN,
		},
		{
			name:           "Not in PMNT domain",
			code:           "ACMT.MCOP.INTR",
			expectedScheme: models.PAYMENT_SCHEME_OTHER,
			expectedType:   models.PAYMENT_TYPE_OTHER,
		},
		{
			name:           "Customer card transaction with cash deposit",
			code:           "PMNT.CCRD.CDPT",
			expectedScheme: models.PAYMENT_SCHEME_OTHER,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Merchant card transaction with POS payment",
			code:           "PMNT.MCRD.POSC",
			expectedScheme: models.PAYMENT_SCHEME_OTHER,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Issued cheques",
			code:           "PMNT.ICHQ.CCHQ",
			expectedScheme: models.PAYMENT_SCHEME_A2A,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Received cheques",
			code:           "PMNT.RCHQ.CCHQ",
			expectedScheme: models.PAYMENT_SCHEME_A2A,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Received cheques unpaid",
			code:           "PMNT.RCHQ.UPCQ",
			expectedScheme: models.PAYMENT_SCHEME_A2A,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Drafts",
			code:           "PMNT.DRFT.STAM",
			expectedScheme: models.PAYMENT_SCHEME_A2A,
			expectedType:   models.PAYMENT_TYPE_TRANSFER,
		},
		{
			name:           "Issued b2b direct debit",
			code:           "PMNT.IDDT.BBDD",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_DEBIT,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Issued core direct debit",
			code:           "PMNT.IDDT.ESDD",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_DEBIT,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Issued direct debits reversal",
			code:           "PMNT.IDDT.PRDD",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_DEBIT,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Received b2b direct debit",
			code:           "PMNT.RDDT.BBDD",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_DEBIT,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Received core direct debit",
			code:           "PMNT.RDDT.ESDD",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_DEBIT,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Received direct debits unpaid",
			code:           "PMNT.RDDT.UPDD",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_DEBIT,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Issued credit transfers SEPA",
			code:           "PMNT.ICDT.ESCT",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_CREDIT,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Received credit transfers SEPA",
			code:           "PMNT.RCDT.ESCT",
			expectedScheme: models.PAYMENT_SCHEME_SEPA_CREDIT,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Received credit transfers reversal",
			code:           "PMNT.RCDT.RRTN",
			expectedScheme: models.PAYMENT_SCHEME_A2A,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
		{
			name:           "Zero balancing treasury management",
			code:           "CAMT.ACCB.ZABA",
			expectedScheme: models.PAYMENT_SCHEME_A2A,
			expectedType:   models.PAYMENT_TYPE_TRANSFER,
		},
		{
			name:           "Miscellaneous credit",
			code:           "PMNT.MCOP.OTHR",
			expectedScheme: models.PAYMENT_SCHEME_OTHER,
			expectedType:   models.PAYMENT_TYPE_PAYIN,
		},
		{
			name:           "Miscellaneous debit",
			code:           "PMNT.MDOP.IADD",
			expectedScheme: models.PAYMENT_SCHEME_OTHER,
			expectedType:   models.PAYMENT_TYPE_PAYOUT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, paymentType := bankingbridge.PaymentSchemeAndType(tt.code)
			assert.Equal(t, tt.expectedScheme, scheme)
			assert.Equal(t, tt.expectedType, paymentType)
		})
	}
}

func TestPaymentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		code           string
		isReversal     bool
		expectedStatus models.PaymentStatus
	}{
		{
			name:           "invalid code format",
			code:           "PMNT.RCDT",
			expectedStatus: models.PAYMENT_STATUS_SUCCEEDED,
		},
		{
			name:           "invalid code format reversal",
			code:           "",
			isReversal:     true,
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "non-PMNT domain",
			code:           "ACMT.MCOP.INTR",
			expectedStatus: models.PAYMENT_STATUS_OTHER,
		},
		{
			name:           "reversal return",
			code:           "PMNT.RCDT.RRTN",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "crossborder return",
			code:           "PMNT.RCDT.XRTN",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "direct debit reversal",
			code:           "PMNT.IDDT.PRDD",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "direct debit unpaid",
			code:           "PMNT.IDDT.UPDD",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "reversal payment cancellation",
			code:           "PMNT.ICDT.RPCR",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "reimbursement",
			code:           "PMNT.CCRD.RIMB",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "unpaid dishonored draft",
			code:           "PMNT.DRFT.UDFT",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "unpaid cheque",
			code:           "PMNT.RCHQ.UPCQ",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "unpaid card transaction",
			code:           "PMNT.CCRD.UPCT",
			expectedStatus: models.PAYMENT_STATUS_REFUNDED,
		},
		{
			name:           "debit adjustment",
			code:           "PMNT.MDOP.DAJT",
			expectedStatus: models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
		},
		{
			name:           "normal SEPA credit transfer",
			code:           "PMNT.RCDT.ESCT",
			expectedStatus: models.PAYMENT_STATUS_SUCCEEDED,
		},
		{
			name:           "normal direct debit",
			code:           "PMNT.IDDT.BBDD",
			expectedStatus: models.PAYMENT_STATUS_SUCCEEDED,
		},
		{
			name:           "Zero balancing treasury management",
			code:         "CAMT.ACCB.ZABA",
			expectedStatus: models.PAYMENT_STATUS_SUCCEEDED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := bankingbridge.PaymentStatus(tt.code, tt.isReversal)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}
