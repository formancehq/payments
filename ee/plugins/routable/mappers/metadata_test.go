package mappers

import (
	"testing"

	"github.com/formancehq/payments/ee/plugins/routable/client"
)

// TestPayableMetadata_AliasKeys locks in the correlation contract
// documented in MAPPINGS.md §5.5: every synced Payment must carry the
// Routable payable UUID under MetadataKeyRoutablePayableID, and the
// originating PI reference under MetadataKeyPaymentInitiationReference
// (when external_id is present).
func TestPayableMetadata_AliasKeys(t *testing.T) {
	cases := []struct {
		name        string
		payable     client.Payable
		wantPIRef   string // expected value for payment_initiation_reference (empty = absent)
		wantPayable string // expected value for payable_id
	}{
		{
			name: "with external_id (Formance-initiated)",
			payable: client.Payable{
				ID:         "pa_abc123",
				ExternalID: "pi_xyz",
				Type:       "ach",
			},
			wantPIRef:   "pi_xyz",
			wantPayable: "pa_abc123",
		},
		{
			name: "without external_id (Routable UI-initiated)",
			payable: client.Payable{
				ID:   "pa_def456",
				Type: "ach",
			},
			wantPIRef:   "", // absent — stripEmpty drops empty values
			wantPayable: "pa_def456",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := PayableMetadata(tc.payable)
			if got := m[MetadataKeyRoutablePayableID]; got != tc.wantPayable {
				t.Errorf("payable_id = %q, want %q", got, tc.wantPayable)
			}
			if got := m[MetadataKeyPaymentInitiationReference]; got != tc.wantPIRef {
				t.Errorf("payment_initiation_reference = %q, want %q", got, tc.wantPIRef)
			}
			// external_id legacy key must remain present alongside the
			// alias for backwards compatibility with anything reading
			// the Routable wire term.
			if got := m[MetadataKeyExternalID]; got != tc.wantPIRef {
				t.Errorf("external_id = %q, want %q (must equal alias)", got, tc.wantPIRef)
			}
		})
	}
}

// TestReceivableMetadata_AliasKeys mirrors the payable contract for the
// inbound path so the correlation works regardless of payment direction.
func TestReceivableMetadata_AliasKeys(t *testing.T) {
	cases := []struct {
		name        string
		receivable  client.Receivable
		wantPIRef   string
		wantPayable string
	}{
		{
			name: "with external_id",
			receivable: client.Receivable{
				ID:         "re_abc",
				ExternalID: "pi_inbound",
				Type:       "ach",
			},
			wantPIRef:   "pi_inbound",
			wantPayable: "re_abc",
		},
		{
			name: "without external_id",
			receivable: client.Receivable{
				ID:   "re_def",
				Type: "ach",
			},
			wantPIRef:   "",
			wantPayable: "re_def",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := ReceivableMetadata(tc.receivable)
			if got := m[MetadataKeyRoutablePayableID]; got != tc.wantPayable {
				t.Errorf("payable_id = %q, want %q", got, tc.wantPayable)
			}
			if got := m[MetadataKeyPaymentInitiationReference]; got != tc.wantPIRef {
				t.Errorf("payment_initiation_reference = %q, want %q", got, tc.wantPIRef)
			}
			if got := m[MetadataKeyExternalID]; got != tc.wantPIRef {
				t.Errorf("external_id = %q, want %q (must equal alias)", got, tc.wantPIRef)
			}
		})
	}
}
