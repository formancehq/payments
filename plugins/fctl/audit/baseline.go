package audit

import "sort"

// BaselineRevision is the legacy fctl commit this baseline was transcribed
// from. Every SourceFile below is a path in that tree.
const BaselineRevision = "693c58e27865f83332e6c3199d61fed81b742f41"

// BaselineCommand is one executable legacy fctl `payments` command.
//
// Grouping-only cobra commands (`payments`, `payments pools`, …) and shared
// helpers (`cmd/payments/versions`, `cmd/payments/connectors/views`,
// `cmd/payments/connectors/internal`) are not commands and are not listed: the
// baseline counts executable leaves only.
type BaselineCommand struct {
	// Path is the canonical invocation, without the `fctl` prefix.
	Path string
	// SourceFile is the declaring file at BaselineRevision.
	SourceFile string
	// LegacyOps are the operationIds the legacy command reaches, in the order
	// the legacy code issues them. Both the V1- and V3-namespace calls the
	// legacy command can make are listed, because legacy commands branch on a
	// probed server version.
	LegacyOps []string
	// V3Ops are the current /v3 operationIds that carry the same behaviour, in
	// issue order. Empty means the behaviour has no faithful v3 equivalent and
	// Exclusion states why.
	V3Ops []string
	// Exclusion is the evidence-backed reason the command is not carried over.
	// Non-empty exactly when V3Ops is empty.
	Exclusion string
}

// Baseline is the complete set of executable legacy fctl `payments` commands at
// BaselineRevision, with each one's mapping onto the current /v3 surface.
var Baseline = []BaselineCommand{
	// accounts
	{
		Path:       "payments accounts list",
		SourceFile: "cmd/payments/accounts/list.go",
		LegacyOps:  []string{"listAccounts"},
		V3Ops:      []string{"v3ListAccounts"},
	},
	{
		Path:       "payments accounts get <accountID>",
		SourceFile: "cmd/payments/accounts/show.go",
		LegacyOps:  []string{"getAccount"},
		V3Ops:      []string{"v3GetAccount"},
	},
	{
		Path:       "payments accounts create <file>|-",
		SourceFile: "cmd/payments/accounts/create.go",
		LegacyOps:  []string{"createAccount"},
		V3Ops:      []string{"v3CreateAccount"},
	},
	{
		Path:       "payments accounts balances <accountID>",
		SourceFile: "cmd/payments/accounts/balances.go",
		LegacyOps:  []string{"getAccountBalances"},
		V3Ops:      []string{"v3GetAccountBalances"},
	},

	// bank accounts
	{
		Path:       "payments bank_accounts list",
		SourceFile: "cmd/payments/bankaccounts/list.go",
		LegacyOps:  []string{"listBankAccounts", "v3ListBankAccounts"},
		V3Ops:      []string{"v3ListBankAccounts"},
	},
	{
		Path:       "payments bank_accounts get <bankAccountID>",
		SourceFile: "cmd/payments/bankaccounts/show.go",
		LegacyOps:  []string{"getBankAccount", "v3GetBankAccount"},
		V3Ops:      []string{"v3GetBankAccount"},
	},
	{
		Path:       "payments bank_accounts create <file>|-",
		SourceFile: "cmd/payments/bankaccounts/create.go",
		LegacyOps:  []string{"createBankAccount", "v3CreateBankAccount"},
		V3Ops:      []string{"v3CreateBankAccount"},
	},
	{
		Path:       "payments bank_accounts forward <bankAccountID> <connectorID>",
		SourceFile: "cmd/payments/bankaccounts/forward.go",
		LegacyOps:  []string{"forwardBankAccount", "v3ForwardBankAccount"},
		V3Ops:      []string{"v3ForwardBankAccount"},
	},
	{
		Path:       "payments bank_accounts update-metadata <bankAccountID> [<key>=<value>...]",
		SourceFile: "cmd/payments/bankaccounts/update_metadata.go",
		LegacyOps:  []string{"updateBankAccountMetadata", "v3UpdateBankAccountMetadata"},
		V3Ops:      []string{"v3UpdateBankAccountMetadata"},
	},

	// connectors
	{
		Path:       "payments connectors list",
		SourceFile: "cmd/payments/connectors/list.go",
		LegacyOps:  []string{"listAllConnectors", "v3ListConnectors"},
		V3Ops:      []string{"v3ListConnectors"},
	},
	{
		Path:       "payments connectors list-available",
		SourceFile: "cmd/payments/connectors/configs.go",
		LegacyOps:  []string{"listConfigsAvailableConnectors", "v3ListConnectorConfigs"},
		V3Ops:      []string{"v3ListConnectorConfigs"},
	},
	{
		Path:       "payments connectors install <connector> <file>|-",
		SourceFile: "cmd/payments/connectors/install/install.go",
		LegacyOps:  []string{"v3ListConnectorConfigs", "installConnector"},
		V3Ops:      []string{"v3ListConnectorConfigs", "v3InstallConnector"},
	},
	{
		Path:       "payments connectors uninstall",
		SourceFile: "cmd/payments/connectors/uninstall.go",
		LegacyOps:  []string{"uninstallConnector", "uninstallConnectorV1", "v3UninstallConnector"},
		V3Ops:      []string{"v3UninstallConnector"},
	},
	{
		Path:       "payments connectors get-config",
		SourceFile: "cmd/payments/connectors/configs/getconfig.go",
		LegacyOps:  []string{"listAllConnectors", "readConnectorConfig", "readConnectorConfigV1"},
		V3Ops:      []string{"v3ListConnectors", "v3GetConnectorConfig"},
	},
	{
		Path:       "payments connectors update-config <connector> <file>|-",
		SourceFile: "cmd/payments/connectors/configs/updateconfig.go",
		LegacyOps:  []string{"v3ListConnectorConfigs", "updateConnectorConfigV1"},
		V3Ops:      []string{"v3ListConnectorConfigs", "v3UpdateConnectorConfig"},
	},

	// connector schedules
	{
		Path:       "payments connectors schedules list <connectorID>",
		SourceFile: "cmd/payments/connectors/schedules/list.go",
		LegacyOps:  []string{"v3ListConnectorSchedules"},
		V3Ops:      []string{"v3ListConnectorSchedules"},
	},
	{
		Path:       "payments connectors schedules get <connectorID> <scheduleID>",
		SourceFile: "cmd/payments/connectors/schedules/show.go",
		LegacyOps:  []string{"v3GetConnectorSchedule"},
		V3Ops:      []string{"v3GetConnectorSchedule"},
	},
	{
		Path:       "payments connectors schedules instances list <connectorID> <scheduleID>",
		SourceFile: "cmd/payments/connectors/schedules/instances/list.go",
		LegacyOps:  []string{"v3ListConnectorScheduleInstances"},
		V3Ops:      []string{"v3ListConnectorScheduleInstances"},
	},

	// payments
	{
		Path:       "payments payments list",
		SourceFile: "cmd/payments/payments/list.go",
		LegacyOps:  []string{"listPayments"},
		V3Ops:      []string{"v3ListPayments"},
	},
	{
		Path:       "payments payments get <paymentID>",
		SourceFile: "cmd/payments/payments/show.go",
		LegacyOps:  []string{"getPayment"},
		V3Ops:      []string{"v3GetPayment"},
	},
	{
		Path:       "payments payments create <file>|-",
		SourceFile: "cmd/payments/payments/create.go",
		LegacyOps:  []string{"createPayment"},
		V3Ops:      []string{"v3CreatePayment"},
	},
	{
		Path:       "payments payments set-metadata <paymentID> [<key>=<value>...]",
		SourceFile: "cmd/payments/payments/set_metadata.go",
		LegacyOps:  []string{"updateMetadata"},
		V3Ops:      []string{"v3UpdatePaymentMetadata"},
	},

	// transfer initiations -> payment initiations
	{
		Path:       "payments transfer_initiation list",
		SourceFile: "cmd/payments/transferinitiation/list.go",
		LegacyOps:  []string{"listTransferInitiations"},
		V3Ops:      []string{"v3ListPaymentInitiations"},
	},
	{
		Path:       "payments transfer_initiation get <transferID>",
		SourceFile: "cmd/payments/transferinitiation/show.go",
		LegacyOps:  []string{"getTransferInitiation"},
		V3Ops:      []string{"v3GetPaymentInitiation"},
	},
	{
		Path:       "payments transfer_initiation create <file>|-",
		SourceFile: "cmd/payments/transferinitiation/create.go",
		LegacyOps:  []string{"createTransferInitiation"},
		V3Ops:      []string{"v3InitiatePayment"},
	},
	{
		Path:       "payments transfer_initiation delete <transferID>",
		SourceFile: "cmd/payments/transferinitiation/delete.go",
		LegacyOps:  []string{"deleteTransferInitiation"},
		V3Ops:      []string{"v3DeletePaymentInitiation"},
	},
	{
		Path:       "payments transfer_initiation approve <transferInitiationID>",
		SourceFile: "cmd/payments/transferinitiation/approve.go",
		LegacyOps:  []string{"v3ApprovePaymentInitiation"},
		V3Ops:      []string{"v3ApprovePaymentInitiation"},
	},
	{
		Path:       "payments transfer_initiation reject <transferInitiationID>",
		SourceFile: "cmd/payments/transferinitiation/reject.go",
		LegacyOps:  []string{"v3RejectPaymentInitiation"},
		V3Ops:      []string{"v3RejectPaymentInitiation"},
	},
	{
		Path:       "payments transfer_initiation retry <transferID>",
		SourceFile: "cmd/payments/transferinitiation/retry.go",
		LegacyOps:  []string{"retryTransferInitiation"},
		V3Ops:      []string{"v3RetryPaymentInitiation"},
	},
	{
		Path:       "payments transfer_initiation reverse <transferID> <file>|-",
		SourceFile: "cmd/payments/transferinitiation/reverse.go",
		LegacyOps:  []string{"reverseTransferInitiation"},
		V3Ops:      []string{"v3ReversePaymentInitiation"},
	},
	{
		Path:       "payments transfer_initiation update_status <transferID> <status>",
		SourceFile: "cmd/payments/transferinitiation/update_status.go",
		LegacyOps:  []string{"updateTransferInitiationStatus"},
		Exclusion: "No faithful v3 equivalent, and no capability loss. " +
			"`UpdateTransferInitiationStatusRequest.status` accepts exactly " +
			"{VALIDATED, REJECTED} (openapi.yaml, components.schemas." +
			"UpdateTransferInitiationStatusRequest), which v3 splits into the " +
			"dedicated v3ApprovePaymentInitiation and v3RejectPaymentInitiation " +
			"operations already carried by `transfer_initiation approve` and " +
			"`transfer_initiation reject`. The legacy command labels itself " +
			"\"deprecated in >= v3.0.0\" at cmd/payments/transferinitiation/" +
			"update_status.go:47. Re-adding a status verb on v3 would require " +
			"choosing an operation per status value, which is exactly what the " +
			"two dedicated commands already express.",
	},

	// pools
	{
		Path:       "payments pools list",
		SourceFile: "cmd/payments/pools/list.go",
		LegacyOps:  []string{"listPools"},
		V3Ops:      []string{"v3ListPools"},
	},
	{
		Path:       "payments pools get <poolID>",
		SourceFile: "cmd/payments/pools/show.go",
		LegacyOps:  []string{"getPool"},
		V3Ops:      []string{"v3GetPool"},
	},
	{
		Path:       "payments pools create <file>|-",
		SourceFile: "cmd/payments/pools/create.go",
		LegacyOps:  []string{"createPool", "v3CreatePool"},
		V3Ops:      []string{"v3CreatePool"},
	},
	{
		Path:       "payments pools delete <poolID>",
		SourceFile: "cmd/payments/pools/delete.go",
		LegacyOps:  []string{"deletePool"},
		V3Ops:      []string{"v3DeletePool"},
	},
	{
		Path:       "payments pools update-query <poolID> <file>|-",
		SourceFile: "cmd/payments/pools/update_query.go",
		LegacyOps:  []string{"v3UpdatePoolQuery"},
		V3Ops:      []string{"v3UpdatePoolQuery"},
	},
	{
		Path:       "payments pools add-account <poolID> <accountID>",
		SourceFile: "cmd/payments/pools/add_accounts.go",
		LegacyOps:  []string{"addAccountToPool"},
		V3Ops:      []string{"v3AddAccountToPool"},
	},
	{
		Path:       "payments pools remove-account <poolID> <accountID>",
		SourceFile: "cmd/payments/pools/remove_account.go",
		LegacyOps:  []string{"removeAccountFromPool"},
		V3Ops:      []string{"v3RemoveAccountFromPool"},
	},
	{
		Path:       "payments pools balances <poolID> <at>",
		SourceFile: "cmd/payments/pools/balances.go",
		LegacyOps:  []string{"getPoolBalances"},
		V3Ops:      []string{"v3GetPoolBalances"},
	},
	{
		Path:       "payments pools latest-balances <poolID>",
		SourceFile: "cmd/payments/pools/latest_balances.go",
		LegacyOps:  []string{"getPoolBalancesLatest", "v3GetPoolBalancesLatest"},
		V3Ops:      []string{"v3GetPoolBalancesLatest"},
	},

	// orders, conversions, tasks
	{
		Path:       "payments orders list",
		SourceFile: "cmd/payments/orders/list.go",
		LegacyOps:  []string{"v3ListOrders"},
		V3Ops:      []string{"v3ListOrders"},
	},
	{
		Path:       "payments orders get <orderID>",
		SourceFile: "cmd/payments/orders/show.go",
		LegacyOps:  []string{"v3GetOrder"},
		V3Ops:      []string{"v3GetOrder"},
	},
	{
		Path:       "payments conversions list",
		SourceFile: "cmd/payments/conversions/list.go",
		LegacyOps:  []string{"v3ListConversions"},
		V3Ops:      []string{"v3ListConversions"},
	},
	{
		Path:       "payments conversions get <conversionID>",
		SourceFile: "cmd/payments/conversions/show.go",
		LegacyOps:  []string{"v3GetConversion"},
		V3Ops:      []string{"v3GetConversion"},
	},
	{
		Path:       "payments tasks get <taskID>",
		SourceFile: "cmd/payments/tasks/show.go",
		LegacyOps:  []string{"v3GetTask"},
		V3Ops:      []string{"v3GetTask"},
	},
}

// MappedBaseline returns the baseline commands that carry over to /v3.
func MappedBaseline() []BaselineCommand {
	var out []BaselineCommand
	for _, c := range Baseline {
		if len(c.V3Ops) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// ExcludedBaseline returns the baseline commands with no faithful /v3
// equivalent, each carrying its exclusion evidence.
func ExcludedBaseline() []BaselineCommand {
	var out []BaselineCommand
	for _, c := range Baseline {
		if len(c.V3Ops) == 0 {
			out = append(out, c)
		}
	}
	return out
}

// BaselineV3Targets returns the sorted, de-duplicated set of /v3 operationIds
// the baseline maps onto.
func BaselineV3Targets() []string {
	seen := map[string]struct{}{}
	for _, c := range Baseline {
		for _, op := range c.V3Ops {
			seen[op] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for op := range seen {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}
