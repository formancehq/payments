package audit

import "sort"

// Blocker is a recorded reason an operation cannot yet be admitted into the
// plugin catalogue, separate from the verified facts about it.
type Blocker struct {
	// OperationIDs are the operations the blocker applies to.
	OperationIDs []string
	// ID is a stable short handle used in the inventory document.
	ID string
	// Summary states the blocker in one sentence.
	Summary string
	// Evidence names the exact sources the blocker was read from.
	Evidence string
}

// Blockers are the recorded admission blockers for the /v3 surface. Each one
// blocks only the operations it lists; no service-wide or family-wide blocking.
var Blockers = []Blocker{
	{
		ID: "B1-undeclared-scopes",
		OperationIDs: []string{
			"v3ForwardBankAccount",
			"v3GetBankAccount",
			"v3UpdateBankAccountMetadata",
		},
		Summary: "The document declares no security block for these three " +
			"operations, so no exact per-operation scope array can be read for " +
			"them, while the server does require authentication. Until the " +
			"document declares their scopes, an exact-scope catalogue entry " +
			"would have to be invented.",
		Evidence: "openapi/v3/v3-api.yaml: GET /v3/bank-accounts/{bankAccountID}, " +
			"PATCH /v3/bank-accounts/{bankAccountID}/metadata and POST " +
			"/v3/bank-accounts/{bankAccountID}/forward carry no `security:` key, " +
			"unlike every other /v3 operation. " +
			"internal/api/v3/router.go registers all three inside the group " +
			"wrapped by jwt.Middleware(a) (\"Authenticated routes\"), so the " +
			"server rejects unauthenticated calls.",
	},
	{
		ID: "B2-get-with-body",
		OperationIDs: []string{
			"v3ListAccounts",
			"v3ListBankAccounts",
			"v3ListConnectorSchedules",
			"v3ListConnectors",
			"v3ListConversions",
			"v3ListOrders",
			"v3ListPaymentInitiationAdjustments",
			"v3ListPaymentInitiationRelatedPayments",
			"v3ListPaymentInitiations",
			"v3ListPaymentServiceUserConnections",
			"v3ListPaymentServiceUserConnectionsFromConnectorID",
			"v3ListPaymentServiceUserLinkAttemptsFromConnectorID",
			"v3ListPaymentServiceUsers",
			"v3ListPayments",
			"v3ListPools",
		},
		Summary: "These GET operations carry their query in a JSON request " +
			"body (the free-form V3QueryBuilder object). A host transport that " +
			"drops or forbids GET request bodies silently degrades them into " +
			"unfiltered listings, which is a wrong answer rather than an error, " +
			"so the request boundary has to be proven to preserve GET bodies " +
			"before these are admitted.",
		Evidence: "openapi.yaml: each listed operation declares " +
			"`requestBody.content.application/json.schema: V3QueryBuilder` " +
			"alongside method GET; components.schemas.V3QueryBuilder is " +
			"`type: object, additionalProperties: true`. The generated client " +
			"signature matches: pkg/client/v3.go ListAccounts(ctx, pageSize, " +
			"cursor, requestBody map[string]any, …).",
	},
	{
		ID: "B3-unredacted-connector-config",
		OperationIDs: []string{
			"v3GetConnectorConfig",
		},
		Summary: "The read path returns PSP credentials in cleartext and the " +
			"legacy fctl commands printed them verbatim, so admitting this " +
			"operation requires a redaction decision that changes observable " +
			"behaviour relative to the baseline.",
		Evidence: "internal/api/services/connector_configs.go ConnectorsConfig " +
			"re-marshals the stored config unchanged; " +
			"internal/storage/connectors.go ConnectorsGet selects " +
			"pgp_sym_decrypt(config, …) AS decrypted_config. At legacy fctl " +
			BaselineRevision + " the per-connector views print credentials " +
			"directly (for example cmd/payments/connectors/views/stripe.go " +
			"renders config.APIKey) and the tree contains no redaction or " +
			"masking helper.",
	},
}

// BlockedOperationIDs returns the sorted, de-duplicated set of operationIds
// carrying at least one blocker.
func BlockedOperationIDs() []string {
	seen := map[string]struct{}{}
	for _, b := range Blockers {
		for _, id := range b.OperationIDs {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Divergence is a recorded mismatch between the document and the server that is
// not itself an admission blocker but changes what fctl may assume.
type Divergence struct {
	ID       string
	Summary  string
	Evidence string
}

// Divergences are the recorded spec-versus-server mismatches.
var Divergences = []Divergence{
	{
		ID: "D1-info-probe-auth",
		Summary: "The document declares `getServerInfo` (GET /_info) under " +
			"`Authorization: [payments:read]`, but the server registers it on " +
			"the root router outside every authenticated group, so it answers " +
			"unauthenticated. fctl needs the unauthenticated read to learn the " +
			"major before it can pick a provider, so the server behaviour is " +
			"the one to rely on — and the document is the one to fix.",
		Evidence: "openapi.yaml GET /_info declares the security block; " +
			"internal/api/router.go NewRouter calls r.Get(\"/_info\", " +
			"api.InfoHandler(info)) on the root router, and jwt.Middleware is " +
			"applied only inside the per-version builders " +
			"(internal/api/v3/router.go, internal/api/v2/router.go).",
	},
	{
		ID: "D2-scopes-not-enforced-in-service",
		Summary: "The `payments:read` / `payments:write` scope arrays are a " +
			"declared contract, not a payments-service check. The service " +
			"authenticates only; it never inspects scopes. An exact-scope " +
			"catalogue is therefore provable from the document but cannot be " +
			"validated against this service's own behaviour.",
		Evidence: "The repository contains no occurrence of `payments:read` or " +
			"`payments:write` in Go sources, and no scope check under " +
			"internal/api/. internal/api/v3/router.go uses jwt.Middleware(a), " +
			"which in go-libs v5.6.1 pkg/authn/jwt/middleware.go only calls " +
			"Authenticate; scope enforcement (ErrMissingScope, " +
			"CheckEndpointSpecificScopesClaim) lives in ControlPlaneMiddleware, " +
			"which this router does not use.",
	},
	{
		ID: "D3-v1-deprecated-connector-paths",
		Summary: "Five payments.v1 operations are marked deprecated in the " +
			"document; all five are connector paths superseded by " +
			"connectorId-qualified v1 forms and by /v3 operations. The legacy " +
			"fctl commands still reference two of them as version-dependent " +
			"fallbacks, which is behaviour the v3-only plugin does not carry.",
		Evidence: "openapi.yaml marks `deprecated: true` on uninstallConnector, " +
			"readConnectorConfig, resetConnector, listConnectorTasks and " +
			"getConnectorTask. At legacy fctl " + BaselineRevision + ", " +
			"cmd/payments/connectors/uninstall.go references uninstallConnector " +
			"and cmd/payments/connectors/configs/getconfig.go references " +
			"readConnectorConfig.",
	},
	{
		ID: "D4-sdk-name-override-gap",
		Summary: "63 of the 64 /v3 operations carry an " +
			"`x-speakeasy-name-override`; `v3UpdateConnectorConfig` does not. " +
			"Its generated Go method is therefore V3.V3UpdateConnectorConfig " +
			"rather than V3.UpdateConnectorConfig, so an adapter that derives " +
			"the client method from the operationId by convention will get " +
			"exactly this one wrong. Cosmetic in the document, load-bearing in " +
			"an adapter.",
		Evidence: "openapi/v3/v3-api.yaml declares no " +
			"x-speakeasy-name-override under operationId v3UpdateConnectorConfig, " +
			"unlike every sibling operation; pkg/client/v3.go declares " +
			"`func (s *V3) V3UpdateConnectorConfig(...)` while its siblings drop " +
			"the v3 prefix. Asserted by TestDocumentedSDKNameOverrideGap.",
	},
}
