package audit_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/formancehq/payments/plugins/fctl/audit"
)

// specPath is the merged document, relative to this package directory.
const specPath = "../../../openapi.yaml"

func build(t *testing.T) *audit.Report {
	t.Helper()
	report, err := audit.Build(specPath)
	if err != nil {
		t.Fatalf("build report: %v", err)
	}
	return report
}

// TestReportMatchesGolden is the determinism gate: the inventory numbers and
// per-operation facts quoted in the inventory document are exactly what the
// current document yields. A spec change is expected to fail this test.
func TestReportMatchesGolden(t *testing.T) {
	report := build(t)

	got, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("encode report: %v", err)
	}
	got = append(got, '\n')

	golden := filepath.Join("testdata", "report.json")
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("report drifted from %s; run `just fctl-audit` and review the diff", golden)
	}
}

// TestGeneratedMarkdownIsCommitted keeps the generated tables in the docs
// directory in step with the golden report.
func TestGeneratedMarkdownIsCommitted(t *testing.T) {
	report := build(t)

	path := filepath.Join("..", "docs", "v3-operations.generated.md")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated markdown: %v", err)
	}
	if report.Markdown() != string(want) {
		t.Fatalf("%s is out of date; run `just fctl-audit`", path)
	}
}

// TestOperationIDsAreUnique guards the assumption every downstream mapping
// relies on: an operationId identifies exactly one (method, path).
func TestOperationIDsAreUnique(t *testing.T) {
	ops, err := audit.LoadOperations(specPath)
	if err != nil {
		t.Fatalf("load operations: %v", err)
	}

	seen := map[string]audit.Operation{}
	for _, op := range ops {
		if prev, dup := seen[op.OperationID]; dup {
			t.Errorf("operationId %q declared twice: %s %s and %s %s",
				op.OperationID, prev.Method, prev.Path, op.Method, op.Path)
		}
		seen[op.OperationID] = op
	}
	if len(seen) != len(ops) {
		t.Errorf("got %d unique operationIds for %d operations", len(seen), len(ops))
	}
}

// TestTagPartition asserts the two tags partition the document with nothing
// left over, so "the v3 surface" is a well-defined denominator.
func TestTagPartition(t *testing.T) {
	report := build(t)
	tot := report.Totals

	if tot.V1Operations+tot.V3Operations != tot.SpecOperations {
		t.Errorf("v1(%d) + v3(%d) != total(%d)", tot.V1Operations, tot.V3Operations, tot.SpecOperations)
	}
	if tot.UniqueOperationIDs != tot.SpecOperations {
		t.Errorf("unique operationIds %d != operations %d", tot.UniqueOperationIDs, tot.SpecOperations)
	}
	if tot.V3Operations == 0 {
		t.Fatal("no payments.v3 operations found; the spec path or tag vocabulary changed")
	}
}

// TestBaselineIsComplete asserts the pinned legacy baseline is internally
// consistent and fully accounted for: every command is either mapped or
// excluded with a reason, and no command is both.
func TestBaselineIsComplete(t *testing.T) {
	seenPath := map[string]bool{}
	for _, c := range audit.Baseline {
		if c.Path == "" || c.SourceFile == "" {
			t.Errorf("baseline entry with empty Path or SourceFile: %+v", c)
		}
		if seenPath[c.Path] {
			t.Errorf("duplicate baseline command path %q", c.Path)
		}
		seenPath[c.Path] = true

		if len(c.LegacyOps) == 0 {
			t.Errorf("%s: no legacy operationIds recorded", c.Path)
		}
		switch {
		case len(c.V3Ops) > 0 && c.Exclusion != "":
			t.Errorf("%s: mapped and excluded at the same time", c.Path)
		case len(c.V3Ops) == 0 && c.Exclusion == "":
			t.Errorf("%s: excluded without a recorded reason", c.Path)
		}
	}

	if got, want := len(audit.MappedBaseline())+len(audit.ExcludedBaseline()), len(audit.Baseline); got != want {
		t.Errorf("mapped + excluded = %d, want %d", got, want)
	}
}

// TestBaselineTargetsExist asserts every operationId the baseline names is
// present in the current document, so the mapping cannot go stale silently.
func TestBaselineTargetsExist(t *testing.T) {
	report := build(t)

	if missing := report.UnknownBaselineTargets(); len(missing) > 0 {
		t.Errorf("baseline maps onto /v3 operations absent from the document: %v", missing)
	}
	if missing := report.UnknownLegacyOps(); len(missing) > 0 {
		t.Errorf("baseline names legacy operationIds absent from the document: %v", missing)
	}
}

// TestFamilyTableIsExactlyTheV3SurfacePlusProbe asserts the frozen family table
// neither misses a /v3 operation nor invents one. A new spec operation must be
// classified deliberately.
func TestFamilyTableIsExactlyTheV3SurfacePlusProbe(t *testing.T) {
	ops, err := audit.LoadOperations(specPath)
	if err != nil {
		t.Fatalf("load operations: %v", err)
	}

	inSpec := map[string]audit.Operation{}
	for _, op := range ops {
		inSpec[op.OperationID] = op
	}

	var want []string
	for _, op := range ops {
		if op.Tag == audit.TagV3 {
			want = append(want, op.OperationID)
		}
	}
	// The single shared server probe is classified too, and is tagged v1.
	want = append(want, "getServerInfo")
	sort.Strings(want)

	got := audit.ClassifiedOperationIDs()

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("family table does not match the /v3 surface plus the probe\n got:  %v\n want: %v", got, want)
	}

	for _, id := range got {
		if _, ok := inSpec[id]; !ok {
			t.Errorf("family table classifies %q, which the document does not declare", id)
		}
	}
}

// TestFrozenFamiliesCoverEveryBaselineTarget asserts the four frozen families
// are sufficient for the whole legacy baseline: no carried-over command lands
// outside them.
func TestFrozenFamiliesCoverEveryBaselineTarget(t *testing.T) {
	frozen := map[audit.Family]bool{}
	for _, f := range audit.FrozenFamilies {
		frozen[f] = true
	}

	for _, op := range audit.BaselineV3Targets() {
		family, ok := audit.FamilyOf(op)
		if !ok {
			t.Errorf("baseline target %s is unclassified", op)
			continue
		}
		if !frozen[family] {
			t.Errorf("baseline target %s lands in non-frozen family %q", op, family)
		}
	}
}

// TestUndeclaredScopeOperationsCarryReleaseGaps asserts the exact-scope rule:
// operations with no security block remain executable only with an explicit
// empty scope set and a recorded release gap, never with guessed scopes.
func TestUndeclaredScopeOperationsCarryReleaseGaps(t *testing.T) {
	report := build(t)

	gapped := map[string]bool{}
	for _, id := range audit.ReleaseGapOperationIDs() {
		gapped[id] = true
	}

	for _, rec := range report.V3 {
		if rec.HasSecurity {
			if len(rec.Scopes) == 0 {
				t.Errorf("%s declares a security block with no scopes; the exact-scope claim is unreadable", rec.OperationID)
			}
			continue
		}
		if !gapped[rec.OperationID] {
			t.Errorf("%s declares no scopes and carries no release gap", rec.OperationID)
		}
	}
}

// TestGetWithBodyOperationsRemainRiskFacts asserts the generated-client audit
// continues to expose these unusual requests without misreporting the closed
// host-adapter compatibility risk as an active admission blocker.
func TestGetWithBodyOperationsRemainRiskFacts(t *testing.T) {
	report := build(t)
	for _, b := range audit.Blockers {
		if b.ID == "B2-get-with-body" {
			t.Fatal("closed B2-get-with-body remains in active blockers")
		}
	}
	count := 0
	for _, rec := range report.V3 {
		if rec.Risk.GetWithBody {
			count++
		}
		for _, blocker := range rec.Blockers {
			if blocker == "B2-get-with-body" || blocker == "B3-unredacted-connector-config" {
				t.Errorf("%s carries closed blocker %s", rec.OperationID, blocker)
			}
		}
	}
	if count != 15 {
		t.Errorf("GET-with-body risk facts = %d, want 15", count)
	}
}

func TestAdmissionBlockersAreEmptyAndReleaseGapsAreExact(t *testing.T) {
	got := audit.BlockedOperationIDs()
	if len(got) != 0 {
		t.Fatalf("admission blocker IDs = %v, want none", got)
	}
	got = audit.ReleaseGapOperationIDs()
	want := []string{"v3ForwardBankAccount", "v3GetBankAccount", "v3UpdateBankAccountMetadata"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("release-gap operation IDs = %v, want %v", got, want)
	}
}

// TestNoIdempotencyKeyIsDeclared pins the fact behind the idempotence risk
// note: nothing in the document offers a client-supplied dedup key, so a
// retried POST is a second effect.
func TestNoIdempotencyKeyIsDeclared(t *testing.T) {
	ops, err := audit.LoadOperations(specPath)
	if err != nil {
		t.Fatalf("load operations: %v", err)
	}

	for _, op := range ops {
		for _, p := range op.Parameters {
			if strings.Contains(strings.ToLower(p.Name), "idempot") {
				t.Errorf("%s declares parameter %q; audit.NoIdempotencyKey is stale", op.OperationID, p.Name)
			}
		}
	}

	if !audit.NoIdempotencyKey {
		t.Error("audit.NoIdempotencyKey is false but no idempotency parameter was found")
	}
}

// TestGapsBlockersAndDivergencesAreWellFormed keeps all three classifications
// usable: stable unique IDs, a summary, and named evidence.
func TestGapsBlockersAndDivergencesAreWellFormed(t *testing.T) {
	report := build(t)
	present := map[string]bool{}
	for _, rec := range report.V3 {
		present[rec.OperationID] = true
	}

	seen := map[string]bool{}
	for _, b := range audit.Blockers {
		if b.ID == "" || b.Summary == "" || b.Evidence == "" {
			t.Errorf("blocker %+v is missing ID, Summary or Evidence", b)
		}
		if seen[b.ID] {
			t.Errorf("duplicate blocker ID %q", b.ID)
		}
		seen[b.ID] = true

		if len(b.OperationIDs) == 0 {
			t.Errorf("blocker %s blocks nothing; blockers are per operation, never service-wide", b.ID)
		}
		if !sort.StringsAreSorted(b.OperationIDs) {
			t.Errorf("blocker %s operationIds are not sorted", b.ID)
		}
		for _, id := range b.OperationIDs {
			if !present[id] {
				t.Errorf("blocker %s names %q, which is not a current /v3 operation", b.ID, id)
			}
		}
	}
	for _, gap := range audit.ReleaseGaps {
		if gap.ID == "" || gap.Summary == "" || gap.Evidence == "" {
			t.Errorf("release gap %+v is missing ID, Summary or Evidence", gap)
		}
		if seen[gap.ID] {
			t.Errorf("duplicate blocker/release-gap ID %q", gap.ID)
		}
		seen[gap.ID] = true
		if len(gap.OperationIDs) == 0 || !sort.StringsAreSorted(gap.OperationIDs) {
			t.Errorf("release gap %s has empty or unsorted operationIds: %v", gap.ID, gap.OperationIDs)
		}
		for _, id := range gap.OperationIDs {
			if !present[id] {
				t.Errorf("release gap %s names %q, which is not a current /v3 operation", gap.ID, id)
			}
		}
	}

	seen = map[string]bool{}
	for _, d := range audit.Divergences {
		if d.ID == "" || d.Summary == "" || d.Evidence == "" {
			t.Errorf("divergence %+v is missing ID, Summary or Evidence", d)
		}
		if seen[d.ID] {
			t.Errorf("duplicate divergence ID %q", d.ID)
		}
		seen[d.ID] = true
	}
}

// TestSecretBearingOperationsResolveToConnectorConfig asserts the credential
// classification is anchored in the document rather than in a name heuristic:
// each secret-bearing operation really does carry V3ConnectorConfig on the
// classified side.
func TestSecretBearingOperationsResolveToConnectorConfig(t *testing.T) {
	report := build(t)

	const configSchema = "V3ConnectorConfig"
	wantRequest := map[string]bool{"v3InstallConnector": true, "v3UpdateConnectorConfig": true}
	wantResponse := map[string]bool{"v3GetConnectorConfig": true}

	requestBodyResolves := map[string]string{
		"V3InstallConnectorRequest": configSchema,
		"V3UpdateConnectorRequest":  configSchema,
	}

	for _, rec := range report.V3 {
		switch rec.Risk.Secret {
		case audit.SecretInbound:
			if !wantRequest[rec.OperationID] {
				t.Errorf("%s classified secret:request unexpectedly", rec.OperationID)
			}
			if requestBodyResolves[rec.RequestBody] != configSchema {
				t.Errorf("%s carries request body %q, which is not a known %s alias",
					rec.OperationID, rec.RequestBody, configSchema)
			}
		case audit.SecretOutbound:
			if !wantResponse[rec.OperationID] {
				t.Errorf("%s classified secret:response unexpectedly", rec.OperationID)
			}
			if rec.SuccessBody != "V3GetConnectorConfigResponse" {
				t.Errorf("%s returns %q, not V3GetConnectorConfigResponse", rec.OperationID, rec.SuccessBody)
			}
		case audit.SecretNone:
			if wantRequest[rec.OperationID] || wantResponse[rec.OperationID] {
				t.Errorf("%s should be classified as secret-bearing", rec.OperationID)
			}
		}
	}
}

// TestPaginatedOperationsDeclareBothCursorParameters asserts pagination is read
// from the document rather than assumed from an operation name.
func TestPaginatedOperationsDeclareBothCursorParameters(t *testing.T) {
	report := build(t)

	for _, rec := range report.V3 {
		if !rec.Risk.Paginated {
			continue
		}
		var cursor, pageSize bool
		for _, p := range rec.Parameters {
			if p.In != "query" {
				continue
			}
			switch p.Name {
			case "cursor":
				cursor = true
			case "pageSize":
				pageSize = true
			}
		}
		if !cursor || !pageSize {
			t.Errorf("%s marked paginated but declares cursor=%v pageSize=%v as query parameters",
				rec.OperationID, cursor, pageSize)
		}
	}
}

// TestServerProbeIsClassifiedOutsideTheFrozenFamilies asserts the /_info probe
// is recorded as host-owned rather than smuggled into a product family.
func TestServerProbeIsClassifiedOutsideTheFrozenFamilies(t *testing.T) {
	family, ok := audit.FamilyOf("getServerInfo")
	if !ok {
		t.Fatal("getServerInfo is unclassified")
	}
	if family != audit.FamilyServerProbe {
		t.Errorf("getServerInfo classified as %q, want %q", family, audit.FamilyServerProbe)
	}
	for _, f := range audit.FrozenFamilies {
		if f == audit.FamilyServerProbe {
			t.Error("the server probe must not be one of the frozen product families")
		}
	}
}
