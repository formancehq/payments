package audit_test

import (
	"testing"

	"github.com/formancehq/payments/plugins/fctl/audit"
)

// The tests in this file pin the numbers quoted in prose in
// docs/command-inventory.md. The generated tables are already covered by the
// golden report; prose is not, so every figure the document asserts is
// re-derived here. If one of these fails, the document is wrong, not the test.

func TestDocumentedDenominators(t *testing.T) {
	tot := build(t).Totals

	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"operations in document", tot.SpecOperations, 109},
		{"unique operationIds", tot.UniqueOperationIDs, 109},
		{"payments.v1 operations", tot.V1Operations, 45},
		{"payments.v3 operations", tot.V3Operations, 64},
		{"deprecated operations", tot.DeprecatedOperations, 5},
		{"legacy fctl commands", tot.BaselineCommands, 45},
		{"legacy commands mapped", tot.BaselineMapped, 44},
		{"legacy commands excluded", tot.BaselineExcluded, 1},
		{"/v3 reached by baseline", tot.V3WithBaseline, 44},
		{"/v3 without legacy precedent", tot.V3WithoutBaseline, 20},
		{"/v3 blocked", tot.V3Blocked, 19},
		{"/v3 without blocker", tot.V3Admissible, 45},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %d, document says %d", c.name, c.got, c.want)
		}
	}
}

func TestDocumentedFamilySizes(t *testing.T) {
	groups := build(t).ByFamily()

	want := map[audit.Family]struct{ total, reached int }{
		audit.FamilyConnectors:  {12, 9},
		audit.FamilyPayments:    {14, 12},
		audit.FamilyAccounts:    {9, 9},
		audit.FamilyPools:       {14, 14},
		audit.FamilyOpenBanking: {15, 0},
	}

	if len(groups) != len(want) {
		t.Errorf("got %d families, document describes %d", len(groups), len(want))
	}

	var frozenTotal, frozenReached int
	for family, expect := range want {
		records := groups[family]
		if len(records) != expect.total {
			t.Errorf("family %q: got %d operations, document says %d", family, len(records), expect.total)
		}
		var reached int
		for _, r := range records {
			if len(r.BaselineCommands) > 0 {
				reached++
			}
		}
		if reached != expect.reached {
			t.Errorf("family %q: got %d operations reached by the baseline, document says %d",
				family, reached, expect.reached)
		}
		if family != audit.FamilyOpenBanking {
			frozenTotal += len(records)
			frozenReached += reached
		}
	}

	if frozenTotal != 49 {
		t.Errorf("frozen families hold %d operations, document says 49", frozenTotal)
	}
	if frozenReached != 44 {
		t.Errorf("frozen families are reached by %d operations, document says 44", frozenReached)
	}
}

func TestDocumentedScopeDistribution(t *testing.T) {
	report := build(t)

	counts := map[string]int{}
	all := make([]audit.Operation, 0, len(report.V1)+len(report.V3))
	all = append(all, report.V1...)
	for _, rec := range report.V3 {
		all = append(all, rec.Operation)
	}

	for _, op := range all {
		switch {
		case !op.HasSecurity:
			counts["none"]++
		case len(op.Scopes) == 1:
			counts[op.Scopes[0]]++
		default:
			t.Errorf("%s declares %d scopes; the document asserts exactly one per declaring operation",
				op.OperationID, len(op.Scopes))
		}
	}

	for name, want := range map[string]int{"payments:read": 55, "payments:write": 51, "none": 3} {
		if counts[name] != want {
			t.Errorf("scope %q: got %d operations, document says %d", name, counts[name], want)
		}
	}
}

func TestDocumentedRiskCounts(t *testing.T) {
	report := build(t)

	var destructive, deletes, paginated, getWithBody, displayOnce, secretIn, secretOut int
	var paginatedWithoutBody []string

	for _, rec := range report.V3 {
		if rec.Method == "DELETE" {
			deletes++
		}
		if rec.Risk.Destructive {
			destructive++
		}
		if rec.Risk.Paginated {
			paginated++
			if !rec.Risk.GetWithBody {
				paginatedWithoutBody = append(paginatedWithoutBody, rec.OperationID)
			}
		}
		if rec.Risk.GetWithBody {
			getWithBody++
		}
		if rec.Risk.DisplayOnce {
			displayOnce++
		}
		switch rec.Risk.Secret {
		case audit.SecretInbound:
			secretIn++
		case audit.SecretOutbound:
			secretOut++
		}
	}

	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"/v3 DELETE operations", deletes, 7},
		{"destructive operations", destructive, 8},
		{"paginated operations", paginated, 17},
		{"GET-with-body operations", getWithBody, 15},
		{"display-once operations", displayOnce, 2},
		{"secret:request operations", secretIn, 2},
		{"secret:response operations", secretOut, 1},
	} {
		if c.got != c.want {
			t.Errorf("%s: got %d, document says %d", c.name, c.got, c.want)
		}
	}

	if len(paginatedWithoutBody) != 2 ||
		paginatedWithoutBody[0] != "v3GetAccountBalances" ||
		paginatedWithoutBody[1] != "v3ListConnectorScheduleInstances" {
		t.Errorf("paginated operations without a request body: got %v, document says "+
			"[v3GetAccountBalances v3ListConnectorScheduleInstances]", paginatedWithoutBody)
	}
}

// TestDocumentedMultiRequestCommands pins the three legacy commands the
// document calls out as issuing more than one request, in issue order, because
// authorisation binds per request and not per command.
func TestDocumentedMultiRequestCommands(t *testing.T) {
	want := map[string][]string{
		"payments connectors get-config":                         {"v3ListConnectors", "v3GetConnectorConfig"},
		"payments connectors install <connector> <file>|-":       {"v3ListConnectorConfigs", "v3InstallConnector"},
		"payments connectors update-config <connector> <file>|-": {"v3ListConnectorConfigs", "v3UpdateConnectorConfig"},
	}

	multi := map[string][]string{}
	for _, c := range audit.Baseline {
		if len(c.V3Ops) > 1 {
			multi[c.Path] = c.V3Ops
		}
	}

	if len(multi) != len(want) {
		t.Errorf("got %d multi-request commands, document says %d: %v", len(multi), len(want), multi)
	}
	for path, wantOps := range want {
		gotOps, ok := multi[path]
		if !ok {
			t.Errorf("%q is not recorded as a multi-request command", path)
			continue
		}
		if len(gotOps) != len(wantOps) {
			t.Errorf("%q issues %v, document says %v", path, gotOps, wantOps)
			continue
		}
		for i := range wantOps {
			if gotOps[i] != wantOps[i] {
				t.Errorf("%q issues %v, document says %v", path, gotOps, wantOps)
				break
			}
		}
	}
}

// TestDocumentedSDKNameOverrideGap pins divergence D4: exactly one /v3
// operation lacks an SDK name override, so its generated client method keeps
// the operationId verbatim.
func TestDocumentedSDKNameOverrideGap(t *testing.T) {
	report := build(t)

	var withoutOverride []string
	for _, rec := range report.V3 {
		if rec.SDKMethod == rec.OperationID {
			withoutOverride = append(withoutOverride, rec.OperationID)
		}
	}

	if len(withoutOverride) != 1 || withoutOverride[0] != "v3UpdateConnectorConfig" {
		t.Errorf("/v3 operations without an SDK name override: got %v, document says "+
			"[v3UpdateConnectorConfig]", withoutOverride)
	}
}

// TestDocumentedMixedTagCommands pins the claim that twelve legacy commands
// name operationIds from both tags, and that exactly two of those are the
// commands which read a /v3 config schema while mutating through v1 — the two
// whose /v3 mapping is multi-operation.
func TestDocumentedMixedTagCommands(t *testing.T) {
	report := build(t)

	tagOf := map[string]string{}
	for _, op := range report.V1 {
		tagOf[op.OperationID] = op.Tag
	}
	for _, rec := range report.V3 {
		tagOf[rec.OperationID] = rec.Tag
	}

	var mixed, mixedMultiOp int
	for _, c := range audit.Baseline {
		var v1, v3 bool
		for _, op := range c.LegacyOps {
			switch tagOf[op] {
			case audit.TagV1:
				v1 = true
			case audit.TagV3:
				v3 = true
			}
		}
		if !v1 || !v3 {
			continue
		}
		mixed++
		if len(c.V3Ops) > 1 {
			mixedMultiOp++
		}
	}

	if mixed != 12 {
		t.Errorf("got %d legacy commands naming both tags, document says 12", mixed)
	}
	if mixedMultiOp != 2 {
		t.Errorf("got %d mixed-tag commands with a multi-operation /v3 mapping, document says 2", mixedMultiOp)
	}
}
