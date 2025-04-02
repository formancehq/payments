package events

const (
	TopicPayments = "payments"

	EventVersion = "v3"
	EventApp     = "payments"

	EventTypeSavedPool                            = "SAVED_POOL"
	EventTypeDeletePool                           = "DELETED_POOL"
	EventTypeSavedPayments                        = "SAVED_PAYMENT"
	EventTypeSavedAccounts                        = "SAVED_ACCOUNT"
	EventTypeSavedBalances                        = "SAVED_BALANCE"
	EventTypeSavedBankAccount                     = "SAVED_BANK_ACCOUNT"
	EventTypeConnectorReset                       = "CONNECTOR_RESET"
	EventTypeSavedPaymentInitiation               = "SAVED_PAYMENT_INITIATION"
	EventTypeSavedPaymentInitiationAdjustment     = "SAVED_PAYMENT_INITIATION_ADJUSTMENT"
	EventTypeSavedPaymentInitiationRelatedPayment = "SAVED_PAYMENT_INITIATION_RELATED_PAYMENT"
)
