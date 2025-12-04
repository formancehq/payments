package events

const (
	TopicPayments = "payments"

	EventVersion = "v3"
	EventApp     = "payments"

	EventTypeSavedPool                                  = "SAVED_POOL"
	EventTypeDeletePool                                 = "DELETED_POOL"
	EventTypeSavedPayments                              = "SAVED_PAYMENT"
	EventTypeDeletedPayments                            = "DELETED_PAYMENT"
	EventTypeSavedAccounts                              = "SAVED_ACCOUNT"
	EventTypeSavedBalances                              = "SAVED_BALANCE"
	EventTypeSavedBankAccount                           = "SAVED_BANK_ACCOUNT"
	EventTypeConnectorReset                             = "CONNECTOR_RESET"
	EventTypeSavedPaymentInitiation                     = "SAVED_PAYMENT_INITIATION"
	EventTypeSavedPaymentInitiationAdjustment           = "SAVED_PAYMENT_INITIATION_ADJUSTMENT"
	EventTypeSavedPaymentInitiationRelatedPayment       = "SAVED_PAYMENT_INITIATION_RELATED_PAYMENT"
	EventTypeUpdatedTask                                = "UPDATED_TASK"
	EventTypeSavedTrade                                 = "SAVED_TRADE"
	EventTypeDeletedTrade                               = "DELETED_TRADE"
	EventTypeOpenBankingUserLinkStatus                  = "OPEN_BANKING_USER_LINK_STATUS"
	EventTypeOpenBankingUserConnectionDataSynced        = "OPEN_BANKING_USER_CONNECTION_DATA_SYNCED"
	EventTypeOpenBankingUserConnectionPendingDisconnect = "OPEN_BANKING_USER_CONNECTION_PENDING_DISCONNECT"
	EventTypeOpenBankingUserConnectionDisconnected      = "OPEN_BANKING_USER_CONNECTION_DISCONNECTED"
	EventTypeOpenBankingUserConnectionReconnected       = "OPEN_BANKING_USER_CONNECTION_RECONNECTED"
	EventTypeOpenBankingUserDisconnected                = "OPEN_BANKING_USER_DISCONNECTED"
)
