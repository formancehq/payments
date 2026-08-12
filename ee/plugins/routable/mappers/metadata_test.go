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
			assertAliasMetadata(t, m, tc.wantPIRef, tc.wantPayable)
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
			assertAliasMetadata(t, m, tc.wantPIRef, tc.wantPayable)
		})
	}
}

// assertAliasMetadata pins the correlation contract precisely:
//   - payable_id MUST be present and equal to wantPayable.
//   - When wantPIRef != "", payment_initiation_reference and external_id
//     MUST both be present and equal to wantPIRef.
//   - When wantPIRef == "" (Routable-UI-initiated row), both alias and
//     legacy keys MUST be ABSENT (not present-with-empty-string), since
//     stripEmpty drops empty values. CodeRabbit flagged that comparing
//     value-only would let a missing key and an empty-string value both
//     pass — this helper closes the gap.
func assertAliasMetadata(t *testing.T, m map[string]string, wantPIRef, wantPayable string) {
	t.Helper()

	got, ok := m[MetadataKeyRoutablePayableID]
	if !ok {
		t.Fatalf("payable_id alias missing entirely from metadata: %v", m)
	}
	if got != wantPayable {
		t.Errorf("payable_id = %q, want %q", got, wantPayable)
	}

	for _, key := range []string{MetadataKeyPaymentInitiationReference, MetadataKeyExternalID} {
		got, present := m[key]
		switch {
		case wantPIRef == "" && present:
			t.Errorf("%s should be ABSENT when external_id is empty (stripEmpty contract); got %q", key, got)
		case wantPIRef != "" && !present:
			t.Errorf("%s should be present when external_id is set (want %q); not found in %v", key, wantPIRef, m)
		case wantPIRef != "" && got != wantPIRef:
			t.Errorf("%s = %q, want %q", key, got, wantPIRef)
		}
	}
}

// TestParseSendOn covers the TS-540 contract: absent/empty means
// unset, a well-formed YYYY-MM-DD is passed
// through verbatim, and anything else is rejected locally so the caller
// can wrap it as a non-retriable ErrInvalidRequest.
func TestParseSendOn(t *testing.T) {
	cases := []struct {
		name    string
		meta    map[string]string
		want    string // "" means expect nil
		wantErr bool
	}{
		{name: "nil metadata leaves send_on unset", meta: nil},
		{name: "absent key leaves send_on unset", meta: map[string]string{MetadataKeyType: "wire"}},
		{name: "empty value leaves send_on unset", meta: map[string]string{MetadataKeySendOn: ""}},
		{name: "valid date passes through", meta: map[string]string{MetadataKeySendOn: "2026-09-01"}, want: "2026-09-01"},
		{name: "leap day is valid", meta: map[string]string{MetadataKeySendOn: "2028-02-29"}, want: "2028-02-29"},
		{name: "slashes rejected", meta: map[string]string{MetadataKeySendOn: "01/09/2026"}, wantErr: true},
		{name: "unpadded rejected", meta: map[string]string{MetadataKeySendOn: "2026-9-1"}, wantErr: true},
		{name: "out of range month rejected", meta: map[string]string{MetadataKeySendOn: "2026-13-45"}, wantErr: true},
		{name: "non-leap Feb 29 rejected", meta: map[string]string{MetadataKeySendOn: "2027-02-29"}, wantErr: true},
		{name: "rfc3339 rejected", meta: map[string]string{MetadataKeySendOn: "2026-09-01T00:00:00Z"}, wantErr: true},
		{name: "free text rejected", meta: map[string]string{MetadataKeySendOn: "tomorrow"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSendOn(tc.meta)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseSendOn(%v) = %v, want error", tc.meta, got)
				}
				if got != nil {
					t.Errorf("ParseSendOn returned %q alongside an error; must return nil", *got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSendOn(%v) unexpected error: %v", tc.meta, err)
			}
			switch {
			case tc.want == "" && got != nil:
				t.Errorf("ParseSendOn = %q, want nil (send immediately)", *got)
			case tc.want != "" && got == nil:
				t.Errorf("ParseSendOn = nil, want %q", tc.want)
			case tc.want != "" && *got != tc.want:
				t.Errorf("ParseSendOn = %q, want %q", *got, tc.want)
			}
		})
	}
}
