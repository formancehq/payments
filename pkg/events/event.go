package events

const (
	TopicPayments = "payments"

	EventVersion = "v3"
	EventApp     = "payments"

	EventTypeSavedPool                                 = "SAVED_POOL"
	EventTypeDeletePool                                = "DELETED_POOL"
	EventTypeSavedPayments                             = "SAVED_PAYMENT"
	EventTypeDeletedPayments                           = "DELETED_PAYMENT"
	EventTypeSavedAccounts                             = "SAVED_ACCOUNT"
	EventTypeSavedBalances                             = "SAVED_BALANCE"
	EventTypeSavedBankAccount                          = "SAVED_BANK_ACCOUNT"
	EventTypeConnectorReset                            = "CONNECTOR_RESET"
	EventTypeSavedPaymentInitiation                    = "SAVED_PAYMENT_INITIATION"
	EventTypeSavedPaymentInitiationAdjustment          = "SAVED_PAYMENT_INITIATION_ADJUSTMENT"
	EventTypeSavedPaymentInitiationRelatedPayment      = "SAVED_PAYMENT_INITIATION_RELATED_PAYMENT"
	EventTypeUpdatedTask                               = "UPDATED_TASK"
	EventTypeBankBridgeUserLinkStatus                  = "BANK_BRIDGE_USER_LINK_STATUS"
	EventTypeBankBridgeUserConnectionDataSynced        = "BANK_BRIDGE_USER_CONNECTION_DATA_SYNCED"
	EventTypeBankBridgeUserConnectionPendingDisconnect = "BANK_BRIDGE_USER_CONNECTION_PENDING_DISCONNECT"
	EventTypeBankBridgeUserConnectionDisconnected      = "BANK_BRIDGE_USER_CONNECTION_DISCONNECTED"
	EventTypeBankBridgeUserConnectionReconnected       = "BANK_BRIDGE_USER_CONNECTION_RECONNECTED"
	EventTypeBankBridgeUserDisconnected                = "BANK_BRIDGE_USER_DISCONNECTED"
)
