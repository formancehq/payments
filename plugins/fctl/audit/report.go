package audit

import (
	"fmt"
	"path/filepath"
	"sort"
)

// Record is one /v3 operation with everything this audit can prove about it.
type Record struct {
	Operation
	Family Family `json:"family"`
	Risk   Risk   `json:"risk"`
	// BaselineCommands are the legacy fctl command paths that map onto this
	// operation, sorted. Empty means the operation has no legacy precedent.
	BaselineCommands []string `json:"baselineCommands"`
	// Blockers are the blocker IDs recorded against this operation, sorted.
	Blockers []string `json:"blockers"`
}

// Totals are the counts the inventory document quotes. Every one of them is
// derived, never transcribed.
type Totals struct {
	// SpecOperations is every operation in the merged document, all tags.
	SpecOperations int `json:"specOperations"`
	// UniqueOperationIDs must equal SpecOperations.
	UniqueOperationIDs int `json:"uniqueOperationIds"`
	// V1Operations and V3Operations partition SpecOperations by tag.
	V1Operations int `json:"v1Operations"`
	V3Operations int `json:"v3Operations"`
	// DeprecatedOperations is the count marked deprecated, all tags.
	DeprecatedOperations int `json:"deprecatedOperations"`

	// BaselineCommands is the executable legacy fctl `payments` command count.
	BaselineCommands int `json:"baselineCommands"`
	// BaselineMapped and BaselineExcluded partition BaselineCommands.
	BaselineMapped   int `json:"baselineMapped"`
	BaselineExcluded int `json:"baselineExcluded"`

	// V3WithBaseline is the number of /v3 operations reached by the baseline.
	V3WithBaseline int `json:"v3WithBaseline"`
	// V3WithoutBaseline is the /v3 surface that has no legacy precedent.
	V3WithoutBaseline int `json:"v3WithoutBaseline"`
	// V3Blocked is the number of /v3 operations carrying a blocker.
	V3Blocked int `json:"v3Blocked"`
	// V3Admissible is V3Operations minus V3Blocked: proven facts, no recorded
	// blocker. It is not an acceptance claim; the runtime gates are separate.
	V3Admissible int `json:"v3Admissible"`
}

// Report is the whole deterministic inventory.
type Report struct {
	// SpecDocument is the base name of the document the report was built from.
	// Only the base name is recorded so the report is identical whether it is
	// produced from the module root or from a package directory.
	SpecDocument string `json:"specDocument"`
	// BaselineRevision pins the legacy fctl tree the baseline came from.
	BaselineRevision string `json:"baselineRevision"`
	Totals           Totals `json:"totals"`
	// V3 is every payments.v3 operation, sorted by operationId.
	V3 []Record `json:"v3"`
	// V1 is every payments.v1 operation, sorted by operationId. It is kept so
	// exclusion and deprecation evidence stays checkable, not because the
	// plugin targets v1.
	V1 []Operation `json:"v1"`
}

// Build reads the document at specPath and produces the full report.
func Build(specPath string) (*Report, error) {
	ops, err := LoadOperations(specPath)
	if err != nil {
		return nil, err
	}

	byID := Index(ops)
	if len(byID) != len(ops) {
		return nil, fmt.Errorf("document declares %d operations but only %d unique operationIds", len(ops), len(byID))
	}

	commandsByOp := map[string][]string{}
	for _, c := range Baseline {
		for _, op := range c.V3Ops {
			commandsByOp[op] = append(commandsByOp[op], c.Path)
		}
	}
	blockersByOp := map[string][]string{}
	for _, b := range Blockers {
		for _, op := range b.OperationIDs {
			blockersByOp[op] = append(blockersByOp[op], b.ID)
		}
	}

	report := &Report{SpecDocument: filepath.Base(specPath), BaselineRevision: BaselineRevision}

	for _, op := range ops {
		switch op.Tag {
		case TagV1:
			report.V1 = append(report.V1, op)
		case TagV3:
			family, ok := FamilyOf(op.OperationID)
			if !ok {
				return nil, fmt.Errorf("operation %s is not classified into a family", op.OperationID)
			}
			commands := append([]string(nil), commandsByOp[op.OperationID]...)
			sort.Strings(commands)
			blocks := append([]string(nil), blockersByOp[op.OperationID]...)
			sort.Strings(blocks)
			report.V3 = append(report.V3, Record{
				Operation:        op,
				Family:           family,
				Risk:             RiskOf(op),
				BaselineCommands: commands,
				Blockers:         blocks,
			})
		default:
			return nil, fmt.Errorf("operation %s carries unknown tag %q", op.OperationID, op.Tag)
		}
	}

	t := Totals{
		SpecOperations:     len(ops),
		UniqueOperationIDs: len(byID),
		V1Operations:       len(report.V1),
		V3Operations:       len(report.V3),
		BaselineCommands:   len(Baseline),
		BaselineMapped:     len(MappedBaseline()),
		BaselineExcluded:   len(ExcludedBaseline()),
	}
	for _, op := range ops {
		if op.Deprecated {
			t.DeprecatedOperations++
		}
	}
	for _, r := range report.V3 {
		if len(r.BaselineCommands) > 0 {
			t.V3WithBaseline++
		} else {
			t.V3WithoutBaseline++
		}
		if len(r.Blockers) > 0 {
			t.V3Blocked++
		}
	}
	t.V3Admissible = t.V3Operations - t.V3Blocked
	report.Totals = t

	return report, nil
}

// UnknownBaselineTargets returns baseline V3Ops entries that do not exist in
// the document. A non-empty result means the mapping references an operation
// the current source does not have.
func (r *Report) UnknownBaselineTargets() []string {
	present := map[string]struct{}{}
	for _, rec := range r.V3 {
		present[rec.OperationID] = struct{}{}
	}
	var missing []string
	for _, op := range BaselineV3Targets() {
		if _, ok := present[op]; !ok {
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	return missing
}

// UnknownLegacyOps returns baseline LegacyOps entries that do not exist in the
// document under any tag.
func (r *Report) UnknownLegacyOps() []string {
	present := map[string]struct{}{}
	for _, op := range r.V1 {
		present[op.OperationID] = struct{}{}
	}
	for _, rec := range r.V3 {
		present[rec.OperationID] = struct{}{}
	}
	seen := map[string]struct{}{}
	var missing []string
	for _, c := range Baseline {
		for _, op := range c.LegacyOps {
			if _, ok := present[op]; ok {
				continue
			}
			if _, dup := seen[op]; dup {
				continue
			}
			seen[op] = struct{}{}
			missing = append(missing, op)
		}
	}
	sort.Strings(missing)
	return missing
}

// ByFamily groups the /v3 records by family, preserving operationId order.
func (r *Report) ByFamily() map[Family][]Record {
	out := map[Family][]Record{}
	for _, rec := range r.V3 {
		out[rec.Family] = append(out[rec.Family], rec)
	}
	return out
}
