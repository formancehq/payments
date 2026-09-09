package audit

import "sort"

// Family is a frozen operation family of the Payments /v3 surface. The first
// four families are the ones the fctl-v2 programme freezes for this plugin
// (connectors/schedules, payments/payment initiations, accounts/bank accounts,
// pools/orders/conversions/tasks). FamilyOpenBanking holds the /v3
// payment-service-user surface, which exists in v3 but has no legacy fctl
// precedent and is therefore recorded, not frozen into the first tranche.
type Family string

const (
	FamilyConnectors  Family = "connectors/schedules"
	FamilyPayments    Family = "payments/payment-initiations"
	FamilyAccounts    Family = "accounts/bank-accounts"
	FamilyPools       Family = "pools/orders/conversions/tasks"
	FamilyOpenBanking Family = "payment-service-users"
	FamilyServerProbe Family = "server-probe"
)

// FrozenFamilies are the families in scope for the first plugin tranche.
var FrozenFamilies = []Family{FamilyConnectors, FamilyPayments, FamilyAccounts, FamilyPools}

// familyOf assigns every /v3 operationId to exactly one family, plus the single
// shared server probe. The table is explicit rather than prefix-derived so that
// a new spec operation fails the completeness test instead of being silently
// absorbed by a prefix rule.
var familyOf = map[string]Family{
	// accounts / bank accounts
	"v3ListAccounts":              FamilyAccounts,
	"v3CreateAccount":             FamilyAccounts,
	"v3GetAccount":                FamilyAccounts,
	"v3GetAccountBalances":        FamilyAccounts,
	"v3ListBankAccounts":          FamilyAccounts,
	"v3CreateBankAccount":         FamilyAccounts,
	"v3GetBankAccount":            FamilyAccounts,
	"v3UpdateBankAccountMetadata": FamilyAccounts,
	"v3ForwardBankAccount":        FamilyAccounts,

	// connectors / schedules
	"v3ListConnectors":                 FamilyConnectors,
	"v3InstallConnector":               FamilyConnectors,
	"v3UninstallConnector":             FamilyConnectors,
	"v3ResetConnector":                 FamilyConnectors,
	"v3ListConnectorConfigs":           FamilyConnectors,
	"v3GetConnectorConfig":             FamilyConnectors,
	"v3UpdateConnectorConfig":          FamilyConnectors,
	"v3ListConnectorCapabilities":      FamilyConnectors,
	"v3GetConnectorCapabilities":       FamilyConnectors,
	"v3ListConnectorSchedules":         FamilyConnectors,
	"v3GetConnectorSchedule":           FamilyConnectors,
	"v3ListConnectorScheduleInstances": FamilyConnectors,

	// payments / payment initiations
	"v3ListPayments":                         FamilyPayments,
	"v3CreatePayment":                        FamilyPayments,
	"v3GetPayment":                           FamilyPayments,
	"v3UpdatePaymentMetadata":                FamilyPayments,
	"v3ListPaymentInitiations":               FamilyPayments,
	"v3InitiatePayment":                      FamilyPayments,
	"v3GetPaymentInitiation":                 FamilyPayments,
	"v3DeletePaymentInitiation":              FamilyPayments,
	"v3RetryPaymentInitiation":               FamilyPayments,
	"v3ApprovePaymentInitiation":             FamilyPayments,
	"v3RejectPaymentInitiation":              FamilyPayments,
	"v3ReversePaymentInitiation":             FamilyPayments,
	"v3ListPaymentInitiationAdjustments":     FamilyPayments,
	"v3ListPaymentInitiationRelatedPayments": FamilyPayments,

	// pools / orders / conversions / tasks
	"v3ListPools":             FamilyPools,
	"v3CreatePool":            FamilyPools,
	"v3GetPool":               FamilyPools,
	"v3DeletePool":            FamilyPools,
	"v3UpdatePoolQuery":       FamilyPools,
	"v3AddAccountToPool":      FamilyPools,
	"v3RemoveAccountFromPool": FamilyPools,
	"v3GetPoolBalances":       FamilyPools,
	"v3GetPoolBalancesLatest": FamilyPools,
	"v3ListOrders":            FamilyPools,
	"v3GetOrder":              FamilyPools,
	"v3ListConversions":       FamilyPools,
	"v3GetConversion":         FamilyPools,
	"v3GetTask":               FamilyPools,

	// payment service users (open banking); recorded, not frozen
	"v3ListPaymentServiceUsers":                           FamilyOpenBanking,
	"v3CreatePaymentServiceUser":                          FamilyOpenBanking,
	"v3GetPaymentServiceUser":                             FamilyOpenBanking,
	"v3DeletePaymentServiceUser":                          FamilyOpenBanking,
	"v3ListPaymentServiceUserConnections":                 FamilyOpenBanking,
	"v3DeletePaymentServiceUserConnector":                 FamilyOpenBanking,
	"v3ForwardPaymentServiceUserToProvider":               FamilyOpenBanking,
	"v3CreateLinkForPaymentServiceUser":                   FamilyOpenBanking,
	"v3ListPaymentServiceUserConnectionsFromConnectorID":  FamilyOpenBanking,
	"v3ListPaymentServiceUserLinkAttemptsFromConnectorID": FamilyOpenBanking,
	"v3GetPaymentServiceUserLinkAttemptFromConnectorID":   FamilyOpenBanking,
	"v3DeletePaymentServiceUserConnectionFromConnectorID": FamilyOpenBanking,
	"v3UpdateLinkForPaymentServiceUserOnConnector":        FamilyOpenBanking,
	"v3AddBankAccountToPaymentServiceUser":                FamilyOpenBanking,
	"v3ForwardPaymentServiceUserBankAccount":              FamilyOpenBanking,

	// shared server probe (declared under the payments.v1 tag, served
	// unprefixed, and the only operation fctl needs before it knows the major)
	"getServerInfo": FamilyServerProbe,
}

// FamilyOf returns the frozen family of an operationId, and whether it is
// classified at all.
func FamilyOf(operationID string) (Family, bool) {
	f, ok := familyOf[operationID]
	return f, ok
}

// ClassifiedOperationIDs returns every classified operationId, sorted.
func ClassifiedOperationIDs() []string {
	out := make([]string, 0, len(familyOf))
	for id := range familyOf {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// secretBearing lists the /v3 operations whose request or response body carries
// PSP credentials in cleartext, with the direction.
//
// Evidence: V3InstallConnectorRequest, V3UpdateConnectorRequest and
// V3GetConnectorConfigResponse.data all resolve to V3ConnectorConfig
// (openapi/v3/v3-connectors-config.yaml), whose 23 provider variants declare
// credential fields such as apiKey, apiSecret, clientSecret, privateKey,
// password, passphrase, accessKey, userCertificateKey, secret,
// configurationToken, stagingToken and webhookSharedSecret. The read path is
// not redacted server-side: internal/api/services/connector_configs.go
// unmarshals and re-marshals the stored config unchanged, and
// internal/storage/connectors.go ConnectorsGet selects
// pgp_sym_decrypt(config, …) AS decrypted_config, so the operation returns the
// decrypted credentials verbatim.
var secretBearing = map[string]SecretDirection{
	"v3InstallConnector":      SecretInbound,
	"v3UpdateConnectorConfig": SecretInbound,
	"v3GetConnectorConfig":    SecretOutbound,
}

// SecretDirection says which side of an operation carries credentials.
type SecretDirection string

const (
	SecretNone     SecretDirection = ""
	SecretInbound  SecretDirection = "request"
	SecretOutbound SecretDirection = "response"
)

// displayOnce lists the /v3 operations whose success body contains a value that
// is only obtainable once and must never be persisted or re-rendered from a
// cache: the provider authorisation URLs minted per link attempt
// (V3PaymentServiceUserCreateLinkResponse.link and
// V3PaymentServiceUserUpdateLinkResponse.link).
var displayOnce = map[string]struct{}{
	"v3CreateLinkForPaymentServiceUser":            {},
	"v3UpdateLinkForPaymentServiceUserOnConnector": {},
}

// Risk is the per-operation risk profile, derived from spec facts plus the two
// explicit tables above.
type Risk struct {
	// Destructive is true for operations that remove or reset server state.
	Destructive bool `json:"destructive"`
	// Secret says whether credentials cross the boundary, and in which
	// direction.
	Secret SecretDirection `json:"secret"`
	// DisplayOnce is true when the success body carries a one-shot value.
	DisplayOnce bool `json:"displayOnce"`
	// ReplaySafe is true for HTTP-idempotent methods (GET, PUT, PATCH, DELETE).
	// The Payments v3 document declares no idempotency key on any operation, so
	// POST operations are never replay-safe: see NoIdempotencyKey.
	ReplaySafe bool `json:"replaySafe"`
	// Paginated is true when the operation exposes cursor + pageSize.
	Paginated bool `json:"paginated"`
	// GetWithBody is true when a GET declares a JSON request body, which is the
	// v3 query-builder shape and a portability hazard for any host transport
	// that drops GET bodies.
	GetWithBody bool `json:"getWithBody"`
}

// NoIdempotencyKey records that the pinned Payments document declares no
// Idempotency-Key (or equivalent) parameter on any operation, in any tag.
// Asserted by TestNoIdempotencyKeyIsDeclared.
const NoIdempotencyKey = true

// destructive lists the /v3 operations that remove or reset server state.
// DELETE-method operations are derived; these are the non-DELETE additions.
var destructiveNonDelete = map[string]struct{}{
	"v3ResetConnector": {},
}

// RiskOf derives the risk profile of an operation.
func RiskOf(op Operation) Risk {
	_, resetLike := destructiveNonDelete[op.OperationID]
	_, once := displayOnce[op.OperationID]

	return Risk{
		Destructive: op.Method == "DELETE" || resetLike,
		Secret:      secretBearing[op.OperationID],
		DisplayOnce: once,
		ReplaySafe:  op.Method == "GET" || op.Method == "PUT" || op.Method == "PATCH" || op.Method == "DELETE",
		Paginated:   op.Paginated(),
		GetWithBody: op.Method == "GET" && op.HasRequestBody(),
	}
}
