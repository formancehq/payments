package events

const (
	TopicPayments = "payments"

	EventVersion = "v3"
	EventApp     = "payments"

	V2EventTypeSavedPool                = "SAVED_POOL"
	V2EventTypeDeletePool               = "DELETED_POOL"
	V2EventTypeSavedPayments            = "SAVED_PAYMENT"
	V2EventTypeSavedAccounts            = "SAVED_ACCOUNT"
	V2EventTypeSavedBalances            = "SAVED_BALANCE"
	V2EventTypeSavedBankAccount         = "SAVED_BANK_ACCOUNT"
	V2EventTypeSavedTransferInitiation  = "SAVED_TRANSFER_INITIATION"
	V2EventTypeDeleteTransferInitiation = "DELETED_TRANSFER_INITIATION"
	V2EventTypeConnectorReset           = "CONNECTOR_RESET"

	V3EventTypeSavedPool                            = "V3_SAVED_POOL"
	V3EventTypeDeletePool                           = "V3_DELETED_POOL"
	V3EventTypeSavedPayments                        = "V3_SAVED_PAYMENT"
	V3EventTypeSavedAccounts                        = "V3_SAVED_ACCOUNT"
	V3EventTypeSavedBalances                        = "V3_SAVED_BALANCE"
	V3EventTypeSavedBankAccount                     = "V3_SAVED_BANK_ACCOUNT"
	V3EventTypeConnectorReset                       = "V3_CONNECTOR_RESET"
	V3EventTypeSavedPaymentInitiation               = "V3_SAVED_PAYMENT_INITIATION"
	V3EventTypeSavedPaymentInitiationAdjustment     = "V3_SAVED_PAYMENT_INITIATION_ADJUSTMENT"
	V3EventTypeSavedPaymentInitiationRelatedPayment = "V3_SAVED_PAYMENT_INITIATION_RELATED_PAYMENT"
)
