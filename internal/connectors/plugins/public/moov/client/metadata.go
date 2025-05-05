package client

const (
	moovMetadataSpecNamespace = "com.moov.spec/"

	// Account and wallet related metadata keys
	MoovAccountIDMetadataKey = moovMetadataSpecNamespace + "account_id"
	MoovWalletIDMetadataKey = moovMetadataSpecNamespace + "wallet_id"
	MoovBankAccountIDMetadataKey = moovMetadataSpecNamespace + "bank_account_id"
	
	// Transfer related metadata keys
	MoovTransferTypeMetadataKey = moovMetadataSpecNamespace + "transfer_type"
	MoovTransferStatusMetadataKey = moovMetadataSpecNamespace + "transfer_status"
	MoovSourceTypeMetadataKey = moovMetadataSpecNamespace + "source_type"
	MoovDestinationTypeMetadataKey = moovMetadataSpecNamespace + "destination_type"
	
	// Bank account related metadata keys
	MoovRoutingNumberMetadataKey = moovMetadataSpecNamespace + "routing_number"
	MoovAccountNumberLastFourMetadataKey = moovMetadataSpecNamespace + "account_number_last_four"
	MoovBankAccountTypeMetadataKey = moovMetadataSpecNamespace + "bank_account_type"
	MoovBankAccountHolderNameMetadataKey = moovMetadataSpecNamespace + "bank_account_holder_name"
	MoovBankAccountHolderTypeMetadataKey = moovMetadataSpecNamespace + "bank_account_holder_type"
)