package filters

import (
	"strings"
	"testing"
)

// TestAllowsAcceptsDeclaredFields exercises every declared field with each of
// its declared operators and asserts Allows returns nil. This guards against
// drift between the spec and what the storage queryContext functions accept.
func TestAllowsAcceptsDeclaredFields(t *testing.T) {
	for _, spec := range All() {
		t.Run(spec.Resource, func(t *testing.T) {
			for _, f := range spec.Fields {
				for _, op := range operatorsFor(f) {
					if err := spec.Allows(f.Name, op); err != nil {
						t.Errorf("Allows(%q, %q) = %v; want nil", f.Name, op, err)
					}
				}
			}
		})
	}
}

// TestAllowsRejectsUnknownKey ensures a key not declared in the spec is rejected.
func TestAllowsRejectsUnknownKey(t *testing.T) {
	for _, spec := range All() {
		t.Run(spec.Resource, func(t *testing.T) {
			err := spec.Allows("nonexistent_field_xyz", "$match")
			if err == nil {
				t.Fatalf("Allows(nonexistent, $match) returned nil; want error")
			}
			if !strings.Contains(err.Error(), "unknown key") {
				t.Errorf("error message = %q; want it to contain \"unknown key\"", err.Error())
			}
		})
	}
}

// TestAllowsRejectsDisallowedOperator ensures fields that only accept $match
// reject comparison operators.
func TestAllowsRejectsDisallowedOperator(t *testing.T) {
	for _, spec := range All() {
		t.Run(spec.Resource, func(t *testing.T) {
			for _, f := range spec.Fields {
				if f.Operators&OpComparison != 0 {
					continue // field accepts comparisons; skip
				}
				err := spec.Allows(f.Name, "$gt")
				if err == nil {
					t.Errorf("Allows(%q, $gt) = nil; want error (field is $match-only)", f.Name)
				}
			}
		})
	}
}

// TestNoDuplicateFieldNames guards against typos producing two entries for the
// same field in a single spec.
func TestNoDuplicateFieldNames(t *testing.T) {
	for _, spec := range All() {
		t.Run(spec.Resource, func(t *testing.T) {
			seen := make(map[string]struct{}, len(spec.Fields))
			for _, f := range spec.Fields {
				if _, dup := seen[f.Name]; dup {
					t.Errorf("duplicate field %q in spec %q", f.Name, spec.Resource)
				}
				seen[f.Name] = struct{}{}
			}
		})
	}
}

// TestRefFieldsCarryOpenAPIRef ensures Type=TypeRef fields name a target
// component, otherwise the OpenAPI generator would emit a broken $ref.
func TestRefFieldsCarryOpenAPIRef(t *testing.T) {
	for _, spec := range All() {
		t.Run(spec.Resource, func(t *testing.T) {
			for _, f := range spec.Fields {
				if f.Type == TypeRef && f.OpenAPIRef == "" {
					t.Errorf("field %q has Type=TypeRef but empty OpenAPIRef", f.Name)
				}
			}
		})
	}
}

func operatorsFor(f Field) []string {
	ops := make([]string, 0, 5)
	if f.Operators&OpMatch != 0 {
		ops = append(ops, "$match")
	}
	if f.Operators&OpComparison != 0 {
		ops = append(ops, "$gt", "$gte", "$lt", "$lte")
	}
	return ops
}
