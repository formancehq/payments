package bankingbridge

import (
	"strings"

	"github.com/formancehq/payments/pkg/domain/models"
)

const (
	TransactionCodeDomainPayment        = "PMNT"
	TransactionCodeDomainCashManagement = "CAMT"

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
	TransactionCodeFamilyAccountBalancing                  = "ACCB"

	TransactionCodeSubFamilyZeroBalancing               = "ZABA"
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
	TransactionCodeSubFamilyUnpaidDishonoredDraft       = "UDFT"
	TransactionCodeSubFamilyUnpaidCardTransaction       = "UPCT"
	TransactionCodeSubFamilyDebitAdjustment             = "DAJT"
)

// https://www.cfonb.org/sites/www.cfonb.org/files/documents/Brochure%20CodesOperation%20V5-1_Finalisee_20221123.pdf
func PaymentSchemeAndType(code string) (models.PaymentScheme, models.PaymentType) {
	codeParts := strings.Split(code, ".")
	if len(codeParts) != 3 {
		return models.PAYMENT_SCHEME_UNKNOWN, models.PAYMENT_TYPE_UNKNOWN
	}

	domain := codeParts[0]
	family := codeParts[1]
	subFamily := codeParts[2]

	switch domain {
	case TransactionCodeDomainPayment:
		return getPaymentDomainSchemeAndType(family, subFamily)

	case TransactionCodeDomainCashManagement:
		if family == TransactionCodeFamilyAccountBalancing  && subFamily == TransactionCodeSubFamilyZeroBalancing {
			return models.PAYMENT_SCHEME_A2A, models.PAYMENT_TYPE_TRANSFER
		}
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_OTHER

	// account overdrafts, interest accumulation, fees etc will not be classified as payments
	default:
		return models.PAYMENT_SCHEME_OTHER, models.PAYMENT_TYPE_OTHER
	}
}

func getPaymentDomainSchemeAndType(family string, subFamily string) (models.PaymentScheme, models.PaymentType) {
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

func PaymentStatus(code string, isReversal bool) models.PaymentStatus {
	codeParts := strings.Split(code, ".")
	// some transactions may have proprietary codes that we don't really know how to parse
	if len(codeParts) != 3 {
		if isReversal {
			return models.PAYMENT_STATUS_REFUNDED
		}
		return models.PAYMENT_STATUS_SUCCEEDED
	}

	domain := codeParts[0]
	family := codeParts[1]
	subFamily := codeParts[2]

	switch domain {
	case TransactionCodeDomainPayment:
		return getPaymentDomainStatus(subFamily)
	case TransactionCodeDomainCashManagement:
		if family == TransactionCodeFamilyAccountBalancing  && subFamily == TransactionCodeSubFamilyZeroBalancing {
			return models.PAYMENT_STATUS_SUCCEEDED
		}
		return models.PAYMENT_STATUS_OTHER

		// account overdrafts, interest accumulation, fees etc will not be classified as payments
	default:
		return models.PAYMENT_STATUS_OTHER
	}
}

func getPaymentDomainStatus(subFamily string) models.PaymentStatus {
	switch subFamily {
	case TransactionCodeSubFamilyReversalReturn, TransactionCodeSubFamilyCrossborderReturn,
		TransactionCodeSubFamilyDirectDebitReversal, TransactionCodeSubFamilyDirectDebitUnpaid,
		TransactionCodeSubFamilyReversalPaymentCancellation, TransactionCodeSubFamilyReimbursement,
		TransactionCodeSubFamilyUnpaidDishonoredDraft,
		TransactionCodeSubFamilyUnpaidCheque,
		TransactionCodeSubFamilyUnpaidCardTransaction:
		return models.PAYMENT_STATUS_REFUNDED
	case TransactionCodeSubFamilyDebitAdjustment:
		return models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT
	}
	return models.PAYMENT_STATUS_SUCCEEDED
}
