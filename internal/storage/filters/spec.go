// Package filters declares the source of truth for the filterable fields
// accepted by each v3 list endpoint of the Payments API. The same Spec values
// are consumed by the storage layer (to gate the allow-list of incoming query
// keys) and by tools/compile-filters (to emit the OpenAPI components describing
// the filter shape for SDK and frontend consumers).
package filters

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownField is returned by Spec.Allows when key is not declared on the
// spec. Callers can errors.Is against it to distinguish the rejection cause.
var ErrUnknownField = errors.New("unknown filter field")

// ErrOperatorNotAllowed is returned by Spec.Allows when the operator is not
// permitted for an otherwise-known field.
var ErrOperatorNotAllowed = errors.New("operator not allowed for filter field")

// Operators is a bitmask of the operators a single field accepts inside the
// query DSL ($match, $lt, $gt, ...).
type Operators uint

const (
	// OpMatch corresponds to the "$match" operator.
	OpMatch Operators = 1 << iota
	// OpComparison corresponds to "$gt", "$gte", "$lt", "$lte".
	OpComparison
)

// Type is the OpenAPI-facing type of a filterable field.
type Type int

const (
	// TypeString → {type: string}.
	TypeString Type = iota
	// TypeBigInt → {type: integer, format: bigint}.
	TypeBigInt
	// TypeRef → {$ref: '#/components/schemas/<OpenAPIRef>'} (re-uses an existing component).
	TypeRef
)

// Field describes one filterable property of a resource.
type Field struct {
	// Name is the wire-level filter key (snake_case, as the storage layer expects).
	Name string
	// Type is the OpenAPI shape of the field's value.
	Type Type
	// OpenAPIRef is the component name when Type == TypeRef (e.g. "V3PaymentStatusEnum").
	OpenAPIRef string
	// Operators declares which operators the field accepts.
	Operators Operators
	// Description is shown in the generated OpenAPI component.
	Description string
}

// Spec describes the filterable shape of one v3 list endpoint.
type Spec struct {
	// Resource is the OpenAPI component-name root, e.g. "Accounts" produces
	// V3AccountsFilter and V3AccountsQueryBuilder.
	Resource string
	// EndpointPath is the v3 path the spec applies to, used in descriptions.
	EndpointPath string
	// Fields are the filterable top-level properties.
	Fields []Field
	// AllowMetadata indicates whether "metadata[<key>]" keys are accepted ($match only).
	AllowMetadata bool
	// Internal is true when the corresponding storage list method is not exposed
	// via a user-facing v3 endpoint that accepts a query body — the spec still
	// gates the storage layer but no OpenAPI component is generated for it.
	Internal bool
}

// FieldByName returns the field with the given name, or nil if absent.
func (s Spec) FieldByName(name string) *Field {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// Allows reports whether (key, operator) is permitted by the spec. It returns
// nil on success, or an error wrapping ErrUnknownField / ErrOperatorNotAllowed
// otherwise. Callers must handle the "metadata[<key>]" case before invoking
// Allows — the storage layer matches those via metadataRegex and emits its own
// JSONB containment SQL.
func (s Spec) Allows(key, operator string) error {
	f := s.FieldByName(key)
	if f == nil {
		return fmt.Errorf("unknown key '%s' when building query: %w", key, ErrUnknownField)
	}
	if !f.allowsOperator(operator) {
		return fmt.Errorf("'%s' column can only be used with %s: %w", key, f.allowedOperatorsLabel(), ErrOperatorNotAllowed)
	}
	return nil
}

func (f Field) allowsOperator(op string) bool {
	switch op {
	case "$match":
		return f.Operators&OpMatch != 0
	case "$gt", "$gte", "$lt", "$lte":
		return f.Operators&OpComparison != 0
	default:
		return false
	}
}

func (f Field) allowedOperatorsLabel() string {
	parts := make([]string, 0, 2)
	if f.Operators&OpMatch != 0 {
		parts = append(parts, "$match")
	}
	if f.Operators&OpComparison != 0 {
		parts = append(parts, "$gt, $gte, $lt, $lte")
	}
	if len(parts) == 0 {
		return "(no operators)"
	}
	return strings.Join(parts, ", ")
}

// All returns every spec declared in this package, in deterministic order.
// tools/compile-filters consumes this to emit the OpenAPI YAML.
func All() []Spec {
	return []Spec{
		Accounts,
		BankAccounts,
		Connectors,
		ConnectorSchedules,
		ConnectorScheduleInstances,
		Conversions,
		OpenBankingConnectionAttempts,
		OpenBankingConnections,
		OpenBankingForwardedUsers,
		Orders,
		PaymentInitiationAdjustments,
		PaymentInitiationReversals,
		PaymentInitiations,
		PaymentServiceUsers,
		Payments,
		Pools,
	}
}
