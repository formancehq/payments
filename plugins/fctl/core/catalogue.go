// Package core implements the Payments v3 command provider.
package core

import (
	"sort"
	"strconv"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const (
	requestBytes  int64 = sdk.GeneratedClientHTTPMaxRequestBytes
	responseBytes int64 = sdk.GeneratedClientHTTPMaxResponseBytes
)

var objectSchema = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`)
var arraySchema = []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"array"}`)

type operationSpec struct {
	id, method, path, scope string
	body, paginated         bool
}

type commandSpec struct {
	path       []string
	arguments  []sdk.Argument
	flags      []sdk.Flag
	operations []operationSpec
	mutation   bool
}

func Catalogue() []sdk.Command {
	specs := catalogueSpecs()
	out := make([]sdk.Command, 0, len(specs))
	for _, spec := range specs {
		out = append(out, spec.command())
	}
	return out
}

func (spec commandSpec) command() sdk.Command {
	operations := make([]sdk.OperationPolicy, 0, len(spec.operations))
	paginated := false
	for _, operation := range spec.operations {
		contentTypes := []string(nil)
		if operation.body {
			contentTypes = []string{"application/json"}
		}
		scopes := []string{}
		if operation.scope != "" {
			scopes = []string{operation.scope}
		}
		operations = append(operations, sdk.OperationPolicy{
			ID: operation.id, Service: sdk.ServicePayments, Scopes: scopes,
			HTTP: &sdk.HTTPOperationPolicy{Method: operation.method, GeneratedClient: &sdk.HTTPGeneratedClientPolicy{
				PathTemplate: operation.path, RequestContentTypes: contentTypes, RequestHeaders: []string{"Accept"}, MaxRequestBytes: requestBytes,
				ResponseLimits: sdk.ResponseLimits{MaxMessageBytes: responseBytes, MaxMessages: 1, MaxAggregateBytes: responseBytes},
			}},
		})
		paginated = paginated || operation.paginated
	}
	risk := sdk.RiskRead
	if spec.mutation {
		risk = sdk.RiskMutation
	}
	maxRequests := uint32(len(operations))
	if paginated {
		maxRequests += sdk.DefaultAllPagesMaxPages - 1
	}
	sensitiveInput := false
	for _, operation := range spec.operations {
		if operation.id == "v3InstallConnector" || operation.id == "v3UpdateConnectorConfig" {
			sensitiveInput = true
		}
	}
	artifacts := make([]sdk.InputArtifactSpec, 0, 2)
	for _, argument := range spec.arguments {
		if argument.Name == "input" {
			artifacts = append(artifacts, sdk.InputArtifactSpec{ArgumentName: "input", MediaTypes: []string{"application/json"}, MaxBytes: requestBytes, AllowFile: true, AllowStdin: true, Sensitive: sensitiveInput})
		}
	}
	for _, flag := range spec.flags {
		if flag.Name == "query" {
			artifacts = append(artifacts, sdk.InputArtifactSpec{FlagName: "query", MediaTypes: []string{"application/json"}, MaxBytes: requestBytes, AllowFile: true, AllowStdin: true})
		}
	}
	publicOutputSchema := objectSchema
	rawOutputSchema := objectSchema
	if paginated {
		publicOutputSchema = arraySchema
		rawOutputSchema = arraySchema
	}
	return sdk.Command{
		ID: "payments.v3." + strings.Join(spec.path, "."), ExecutionKind: sdk.ExecutionKindService, AuthMode: sdk.AuthModeCapability,
		Path: spec.path, Target: sdk.TargetRequirement{Kind: sdk.TargetStack}, Summary: "Payments " + strings.Join(spec.path, " "),
		Long: "Execute the Payments v3 operation through the host-owned endpoint, authentication and transport.", Example: strings.Join(spec.path, " ") + " --help",
		Arguments: spec.arguments, Flags: spec.flags, Auth: []sdk.AuthRequirement{{Capability: "auth.stack"}}, Operations: operations,
		Compatibility: []sdk.ServiceCompatibility{{Service: sdk.ServicePayments, Majors: []uint32{3}}}, Risk: risk,
		InputSchema: inputSchema(spec.arguments, spec.flags), RawOutputSchema: rawOutputSchema, PublicOutputSchema: publicOutputSchema,
		InputArtifacts: artifacts, Pagination: sdk.PaginationSpec{Supported: paginated}, OutputMediaType: "application/json",
		ExecutionPolicy: &sdk.CommandExecutionPolicy{MaxHostRequests: maxRequests},
	}
}

func arg(name string) sdk.Argument {
	return sdk.Argument{Name: name, Usage: name, Type: sdk.ArgumentString, Required: true, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}
func repeatedArg(name string) sdk.Argument {
	return sdk.Argument{Name: name, Usage: name + " as key=value", Type: sdk.ArgumentStringArray, Required: true, Repeated: true, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}
func flag(name string, kind sdk.FlagType, required bool) sdk.Flag {
	return sdk.Flag{Name: name, Usage: name, Type: kind, Required: required, Completion: sdk.CompletionSpec{Kind: sdk.CompletionNone}}
}
func op(id, method, path, scope string, body, paginated bool) operationSpec {
	return operationSpec{id: id, method: method, path: path, scope: scope, body: body, paginated: paginated}
}

func catalogueSpecs() []commandSpec {
	r, w := "payments:read", "payments:write"
	listFlags := []sdk.Flag{flag("query", sdk.FlagString, false), flag("page-size", sdk.FlagInt32, false)}
	input := []sdk.Argument{arg("input")}
	return []commandSpec{
		{path: []string{"accounts", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListAccounts", "GET", "/v3/accounts", r, true, true)}},
		{path: []string{"accounts", "get"}, arguments: []sdk.Argument{arg("account-id")}, operations: []operationSpec{op("v3GetAccount", "GET", "/v3/accounts/{accountID}", r, false, false)}},
		{path: []string{"accounts", "create"}, arguments: input, operations: []operationSpec{op("v3CreateAccount", "POST", "/v3/accounts", w, true, false)}, mutation: true},
		{path: []string{"accounts", "balances"}, arguments: []sdk.Argument{arg("account-id")}, flags: []sdk.Flag{flag("asset", sdk.FlagString, false), flag("from-timestamp", sdk.FlagString, false), flag("to-timestamp", sdk.FlagString, false), flag("page-size", sdk.FlagInt32, false)}, operations: []operationSpec{op("v3GetAccountBalances", "GET", "/v3/accounts/{accountID}/balances", r, false, true)}},
		{path: []string{"bank_accounts", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListBankAccounts", "GET", "/v3/bank-accounts", r, true, true)}},
		{path: []string{"bank_accounts", "get"}, arguments: []sdk.Argument{arg("bank-account-id")}, operations: []operationSpec{op("v3GetBankAccount", "GET", "/v3/bank-accounts/{bankAccountID}", "", false, false)}},
		{path: []string{"bank_accounts", "create"}, arguments: input, operations: []operationSpec{op("v3CreateBankAccount", "POST", "/v3/bank-accounts", w, true, false)}, mutation: true},
		{path: []string{"bank_accounts", "forward"}, arguments: []sdk.Argument{arg("bank-account-id"), arg("connector-id")}, operations: []operationSpec{op("v3ForwardBankAccount", "POST", "/v3/bank-accounts/{bankAccountID}/forward", "", true, false)}, mutation: true},
		{path: []string{"bank_accounts", "update-metadata"}, arguments: []sdk.Argument{arg("bank-account-id"), repeatedArg("metadata")}, operations: []operationSpec{op("v3UpdateBankAccountMetadata", "PATCH", "/v3/bank-accounts/{bankAccountID}/metadata", "", true, false)}, mutation: true},
		{path: []string{"connectors", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListConnectors", "GET", "/v3/connectors", r, true, true)}},
		{path: []string{"connectors", "list-available"}, operations: []operationSpec{op("v3ListConnectorConfigs", "GET", "/v3/connectors/configs", r, false, false)}},
		{path: []string{"connectors", "install"}, arguments: []sdk.Argument{arg("connector"), arg("input")}, operations: []operationSpec{op("v3ListConnectorConfigs", "GET", "/v3/connectors/configs", r, false, false), op("v3InstallConnector", "POST", "/v3/connectors/install/{connector}", w, true, false)}, mutation: true},
		{path: []string{"connectors", "uninstall"}, flags: []sdk.Flag{flag("connector-id", sdk.FlagString, true)}, operations: []operationSpec{op("v3UninstallConnector", "DELETE", "/v3/connectors/{connectorID}", w, false, false)}, mutation: true},
		{path: []string{"connectors", "get-config"}, flags: []sdk.Flag{flag("connector-id", sdk.FlagString, true)}, operations: []operationSpec{op("v3ListConnectors", "GET", "/v3/connectors", r, true, false), op("v3GetConnectorConfig", "GET", "/v3/connectors/{connectorID}/config", r, false, false)}},
		{path: []string{"connectors", "update-config"}, arguments: []sdk.Argument{arg("connector"), arg("input")}, flags: []sdk.Flag{flag("connector-id", sdk.FlagString, true)}, operations: []operationSpec{op("v3ListConnectorConfigs", "GET", "/v3/connectors/configs", r, false, false), op("v3UpdateConnectorConfig", "PATCH", "/v3/connectors/{connectorID}/config", w, true, false)}, mutation: true},
		{path: []string{"connectors", "schedules", "list"}, arguments: []sdk.Argument{arg("connector-id")}, flags: listFlags, operations: []operationSpec{op("v3ListConnectorSchedules", "GET", "/v3/connectors/{connectorID}/schedules", r, true, true)}},
		{path: []string{"connectors", "schedules", "get"}, arguments: []sdk.Argument{arg("connector-id"), arg("schedule-id")}, operations: []operationSpec{op("v3GetConnectorSchedule", "GET", "/v3/connectors/{connectorID}/schedules/{scheduleID}", r, false, false)}},
		{path: []string{"connectors", "schedules", "instances", "list"}, arguments: []sdk.Argument{arg("connector-id"), arg("schedule-id")}, flags: []sdk.Flag{flag("page-size", sdk.FlagInt32, false)}, operations: []operationSpec{op("v3ListConnectorScheduleInstances", "GET", "/v3/connectors/{connectorID}/schedules/{scheduleID}/instances", r, false, true)}},
		{path: []string{"payments", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListPayments", "GET", "/v3/payments", r, true, true)}},
		{path: []string{"payments", "get"}, arguments: []sdk.Argument{arg("payment-id")}, operations: []operationSpec{op("v3GetPayment", "GET", "/v3/payments/{paymentID}", r, false, false)}},
		{path: []string{"payments", "create"}, arguments: input, operations: []operationSpec{op("v3CreatePayment", "POST", "/v3/payments", w, true, false)}, mutation: true},
		{path: []string{"payments", "set-metadata"}, arguments: []sdk.Argument{arg("payment-id"), repeatedArg("metadata")}, operations: []operationSpec{op("v3UpdatePaymentMetadata", "PATCH", "/v3/payments/{paymentID}/metadata", w, true, false)}, mutation: true},
		{path: []string{"transfer_initiation", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListPaymentInitiations", "GET", "/v3/payment-initiations", r, true, true)}},
		{path: []string{"transfer_initiation", "get"}, arguments: []sdk.Argument{arg("transfer-id")}, operations: []operationSpec{op("v3GetPaymentInitiation", "GET", "/v3/payment-initiations/{paymentInitiationID}", r, false, false)}},
		{path: []string{"transfer_initiation", "create"}, arguments: input, flags: []sdk.Flag{flag("no-validation", sdk.FlagBool, false)}, operations: []operationSpec{op("v3InitiatePayment", "POST", "/v3/payment-initiations", w, true, false)}, mutation: true},
		{path: []string{"transfer_initiation", "delete"}, arguments: []sdk.Argument{arg("transfer-id")}, operations: []operationSpec{op("v3DeletePaymentInitiation", "DELETE", "/v3/payment-initiations/{paymentInitiationID}", w, false, false)}, mutation: true},
		{path: []string{"transfer_initiation", "approve"}, arguments: []sdk.Argument{arg("transfer-id")}, operations: []operationSpec{op("v3ApprovePaymentInitiation", "POST", "/v3/payment-initiations/{paymentInitiationID}/approve", w, false, false)}, mutation: true},
		{path: []string{"transfer_initiation", "reject"}, arguments: []sdk.Argument{arg("transfer-id")}, operations: []operationSpec{op("v3RejectPaymentInitiation", "POST", "/v3/payment-initiations/{paymentInitiationID}/reject", w, false, false)}, mutation: true},
		{path: []string{"transfer_initiation", "retry"}, arguments: []sdk.Argument{arg("transfer-id")}, operations: []operationSpec{op("v3RetryPaymentInitiation", "POST", "/v3/payment-initiations/{paymentInitiationID}/retry", w, false, false)}, mutation: true},
		{path: []string{"transfer_initiation", "reverse"}, arguments: []sdk.Argument{arg("transfer-id"), arg("input")}, operations: []operationSpec{op("v3ReversePaymentInitiation", "POST", "/v3/payment-initiations/{paymentInitiationID}/reverse", w, true, false)}, mutation: true},
		{path: []string{"pools", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListPools", "GET", "/v3/pools", r, true, true)}},
		{path: []string{"pools", "get"}, arguments: []sdk.Argument{arg("pool-id")}, operations: []operationSpec{op("v3GetPool", "GET", "/v3/pools/{poolID}", r, false, false)}},
		{path: []string{"pools", "create"}, arguments: input, operations: []operationSpec{op("v3CreatePool", "POST", "/v3/pools", w, true, false)}, mutation: true},
		{path: []string{"pools", "delete"}, arguments: []sdk.Argument{arg("pool-id")}, operations: []operationSpec{op("v3DeletePool", "DELETE", "/v3/pools/{poolID}", w, false, false)}, mutation: true},
		{path: []string{"pools", "update-query"}, arguments: []sdk.Argument{arg("pool-id"), arg("input")}, operations: []operationSpec{op("v3UpdatePoolQuery", "PATCH", "/v3/pools/{poolID}/query", w, true, false)}, mutation: true},
		{path: []string{"pools", "add-account"}, arguments: []sdk.Argument{arg("pool-id"), arg("account-id")}, operations: []operationSpec{op("v3AddAccountToPool", "POST", "/v3/pools/{poolID}/accounts/{accountID}", w, false, false)}, mutation: true},
		{path: []string{"pools", "remove-account"}, arguments: []sdk.Argument{arg("pool-id"), arg("account-id")}, operations: []operationSpec{op("v3RemoveAccountFromPool", "DELETE", "/v3/pools/{poolID}/accounts/{accountID}", w, false, false)}, mutation: true},
		{path: []string{"pools", "balances"}, arguments: []sdk.Argument{arg("pool-id"), arg("at")}, operations: []operationSpec{op("v3GetPoolBalances", "GET", "/v3/pools/{poolID}/balances", r, false, false)}},
		{path: []string{"pools", "latest-balances"}, arguments: []sdk.Argument{arg("pool-id")}, operations: []operationSpec{op("v3GetPoolBalancesLatest", "GET", "/v3/pools/{poolID}/balances/latest", r, false, false)}},
		{path: []string{"orders", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListOrders", "GET", "/v3/orders", r, true, true)}},
		{path: []string{"orders", "get"}, arguments: []sdk.Argument{arg("order-id")}, operations: []operationSpec{op("v3GetOrder", "GET", "/v3/orders/{orderID}", r, false, false)}},
		{path: []string{"conversions", "list"}, flags: listFlags, operations: []operationSpec{op("v3ListConversions", "GET", "/v3/conversions", r, true, true)}},
		{path: []string{"conversions", "get"}, arguments: []sdk.Argument{arg("conversion-id")}, operations: []operationSpec{op("v3GetConversion", "GET", "/v3/conversions/{conversionID}", r, false, false)}},
		{path: []string{"tasks", "get"}, arguments: []sdk.Argument{arg("task-id")}, operations: []operationSpec{op("v3GetTask", "GET", "/v3/tasks/{taskID}", r, false, false)}},
	}
}

func inputSchema(arguments []sdk.Argument, flags []sdk.Flag) []byte {
	type field struct {
		typ      string
		required bool
	}
	fields := map[string]field{}
	for _, value := range arguments {
		typ := "string"
		if value.Repeated {
			typ = "array"
		}
		fields[value.Name] = field{typ, value.Required}
	}
	for _, value := range flags {
		typ := "string"
		if value.Type == sdk.FlagBool {
			typ = "boolean"
		} else if value.Type == sdk.FlagInt32 {
			typ = "integer"
		}
		fields[value.Name] = field{typ, value.Required}
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	properties, required := make([]string, 0, len(names)), make([]string, 0, len(names))
	for _, name := range names {
		value := fields[name]
		schema := `{"type":` + strconv.Quote(value.typ)
		if value.typ == "array" {
			schema += `,"items":{"type":"string"}`
			if value.required {
				schema += `,"minItems":1`
			}
		}
		schema += `}`
		properties = append(properties, strconv.Quote(name)+":"+schema)
		if value.required {
			required = append(required, strconv.Quote(name))
		}
	}
	out := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{` + strings.Join(properties, ",") + `}`
	if len(required) > 0 {
		out += `,"required":[` + strings.Join(required, ",") + `]`
	}
	return []byte(out + `,"additionalProperties":false}`)
}
