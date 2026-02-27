package bankingbridge

import (
	"strings"

	"github.com/formancehq/payments/internal/models"
)

const (
	TransactionCodeDomainPayment = "PMNT"

	TransactionCodeFamilyCustomerCardTransactions          = "CCRD"
	TransactionCodeFamilyMerchantCardTransactions          = "MCRD"
	TransactionCodeFamilyIssuedCheques                     = "ICHQ"
	TransactionCodeFamilyReceivedCheques                   = "RCHQ"
	TransactionCodeFamilyCounterTransactions               = "CNTR"
	TransactionCodeFamilyDrafts                            = "DRFT"
	TransactionCodeFamilyIssuedDirectDebits                = "IDDT"
	TransactionCodeFamilyReceivedDirectDebits              = "RDDT"
	TransactionCodeFamilyIssuedCreditTransfers             = "ICDT"
	TransactionCodeFamilyIssuedCashConcentrationTransfer   = "ICCN"
	TransactionCodeFamilyReceivedCreditTransfers           = "RCDT"
	TransactionCodeFamilyReceivedCashConcentrationTransfer = "RCCN"
	TransactionCodeFamilyIssuedRealTimeCreditTransfer      = "IRCT"
	TransactionCodeFamilyReceivedRealTimeCreditTransfer    = "RRCT"
	TransactionCodeFamilyMiscellaneousCredit               = "MCOP"
	TransactionCodeFamilyMiscellaneousDebit                = "MDOP"

	TransactionCodeSubFamilyCashDeposit                 = "CDPT"
	TransactionCodeSubFamilyCashWithdrawl               = "CWDL"
	TransactionCodeSubFamilyReimbursement               = "RIMB"
	TransactionCodeSubFamilyPointOfSalePayment          = "POSC"
	TransactionCodeSubFamilyUnpaidCheque                = "UPCQ"
	TransactionCodeSubFamilyDomesticCreditTransfers     = "DMCT"
	TransactionCodeSubFamilySEPACreditTransfers         = "ESCT"
	TransactionCodeSubFamilyReversalReturn              = "RRTN"
	TransactionCodeSubFamilyCrossborderReturn           = "XRTN"
	TransactionCodeSubFamilyDirectDebitReversal         = "PRDD"
	TransactionCodeSubFamilyDirectDebitUnpaid           = "UPDD"
	TransactionCodeSubFamilyReversalPaymentCancellation = "RPCR"
)

// https://www.cfonb.org/sites/www.cfonb.org/files/documents/Brochure%20CodesOperation%20V5-1_Finalisee_20221123.pdf
func PaymentSchemeAndType(scheme string) (models.PaymentScheme, models.PaymentType) {
	schemeParts := strings.Split(scheme, ".")
	if len(schemeParts) != 3 {
		return models.PAYMENT_SCHEME_UNKNOWN, models.PAYMENT_TYPE_UNKNOWN
	}

	domain := schemeParts[0]
	family := schemeParts[1]
	subFamily := schemeParts[2]

	// account overdrafts, interest accumulation, fees etc will not be classified as payments
	if domain != TransactionCodeDomainPayment {
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_OTHER
	}

	switch family {
	case TransactionCodeFamilyCustomerCardTransactions:
		if subFamily == TransactionCodeSubFamilyCashDeposit || subFamily == TransactionCodeSubFamilyReimbursement {
			return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYIN
		}
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYOUT

	case TransactionCodeFamilyMerchantCardTransactions:
		if subFamily == TransactionCodeSubFamilyPointOfSalePayment {
			return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYIN
		}
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYOUT

	case TransactionCodeFamilyCounterTransactions:
		if subFamily == TransactionCodeSubFamilyCashDeposit {
			return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYIN
		}
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYOUT

	case TransactionCodeFamilyIssuedCheques:
		return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_PAYOUT

	case TransactionCodeFamilyReceivedCheques:
		if subFamily == TransactionCodeSubFamilyUnpaidCheque {
			return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_PAYOUT
		}
		return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_PAYIN

	case TransactionCodeFamilyDrafts:
		return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_TRANSFER

	case TransactionCodeFamilyIssuedDirectDebits:
		if subFamily == TransactionCodeSubFamilyDirectDebitReversal || subFamily == TransactionCodeSubFamilyDirectDebitUnpaid {
			return models.PAYMENT_SCHEME_SEPA_DEBIT, models.PAYMENT_TYPE_PAYOUT
		}
		return models.PAYMENT_SCHEME_SEPA_DEBIT, models.PAYMENT_TYPE_PAYIN

	case TransactionCodeFamilyReceivedDirectDebits:
		if subFamily == TransactionCodeSubFamilyDirectDebitReversal || subFamily == TransactionCodeSubFamilyDirectDebitUnpaid {
			return models.PAYMENT_SCHEME_SEPA_DEBIT, models.PAYMENT_TYPE_PAYIN
		}
		return models.PAYMENT_SCHEME_SEPA_DEBIT, models.PAYMENT_TYPE_PAYOUT

	case TransactionCodeFamilyIssuedCreditTransfers, TransactionCodeFamilyIssuedCashConcentrationTransfer:
		paymentType := models.PAYMENT_TYPE_PAYOUT
		if subFamily == TransactionCodeSubFamilyReversalReturn || subFamily == TransactionCodeSubFamilyCrossborderReturn {
			paymentType = models.PAYMENT_TYPE_PAYIN
		}

		if subFamily == TransactionCodeSubFamilySEPACreditTransfers {
			return models.PAYMENT_SCHEME_SEPA_CREDIT, paymentType
		}
		return models.PAYMENT_SCHEME_A2A, paymentType

	case TransactionCodeFamilyReceivedCreditTransfers, TransactionCodeFamilyReceivedCashConcentrationTransfer:
		paymentType := models.PAYMENT_TYPE_PAYIN
		if subFamily == TransactionCodeSubFamilyReversalReturn || subFamily == TransactionCodeSubFamilyCrossborderReturn {
			paymentType = models.PAYMENT_TYPE_PAYOUT
		}

		if subFamily == TransactionCodeSubFamilySEPACreditTransfers {
			return models.PAYMENT_SCHEME_SEPA_CREDIT, paymentType
		}
		return models.PAYMENT_SCHEME_A2A, paymentType

	case TransactionCodeFamilyIssuedRealTimeCreditTransfer:
		if subFamily == TransactionCodeSubFamilySEPACreditTransfers {
			return models.PAYMENT_SCHEME_SEPA_CREDIT, models.PAYMENT_TYPE_PAYOUT
		}
		// cancellations, reversals, reimbursements
		return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_PAYIN

	case TransactionCodeFamilyReceivedRealTimeCreditTransfer:
		if subFamily == TransactionCodeSubFamilySEPACreditTransfers {
			return models.PAYMENT_SCHEME_SEPA_CREDIT, models.PAYMENT_TYPE_PAYIN
		}
		// cancellations, reversals
		return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_PAYOUT

	case TransactionCodeFamilyMiscellaneousCredit:
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYIN

	case TransactionCodeFamilyMiscellaneousDebit:
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_PAYOUT
	}
	return models.PAYMENT_SCHEME_UNKNOWN, models.PAYMENT_TYPE_UNKNOWN
}
