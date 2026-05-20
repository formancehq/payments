package mappers

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/ee/plugins/routable/client"
	"github.com/formancehq/payments/internal/models"
)

// SettingsAccountToPSPAccount: stable Routable ID → PSPAccount.Reference,
// metadata flattened, DefaultAsset only when currency is recognised, Raw
// preserved.
func TestSettingsAccountToPSPAccount(t *testing.T) {
	created := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	in := client.Account{
		ID:           "acc_42",
		Object:       "Account",
		Type:         "checking",
		Name:         "Operating",
		CurrencyCode: "USD",
		IsValid:      true,
		CreatedAt:    created,
		TypeDetails: client.AccountTypeDetails{
			AvailableAmount: "1000.00",
			AccountType:     "business",
			BankName:        "JPM Chase",
			AccountNumber:   "***2157",
			RoutingNumber:   "021000021",
		},
	}

	got, err := SettingsAccountToPSPAccount(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Reference != "acc_42" {
		t.Errorf("Reference = %q, want acc_42", got.Reference)
	}
	if got.Name == nil || *got.Name != "Operating" {
		t.Errorf("Name = %v, want *Operating", got.Name)
	}
	if got.DefaultAsset == nil || *got.DefaultAsset != "USD/2" {
		t.Errorf("DefaultAsset = %v, want *USD/2", got.DefaultAsset)
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt mismatch: %v", got.CreatedAt)
	}
	if got.Metadata[MetadataPrefix+"bank_name"] != "JPM Chase" {
		t.Errorf("missing bank_name metadata: %v", got.Metadata)
	}
	if got.Metadata[MetadataPrefix+"is_valid"] != "true" {
		t.Errorf("is_valid metadata = %q, want true", got.Metadata[MetadataPrefix+"is_valid"])
	}
	// Raw must contain the original Routable shape verbatim.
	var roundtrip client.Account
	if err := json.Unmarshal(got.Raw, &roundtrip); err != nil {
		t.Fatalf("Raw not valid JSON: %v", err)
	}
	if roundtrip.ID != "acc_42" {
		t.Errorf("Raw round-trip lost ID: %+v", roundtrip)
	}
}

// Empty currency: DefaultAsset stays nil. Routable's FormatAsset returns
// "" only when the input is empty, so this guards the empty-string case
// specifically (the "XYZ unknown ISO" path falls through to "XYZ" rather
// than nil — that's a separate FormatAsset semantic the helper inherits
// from go-libs/currency).
func TestSettingsAccountToPSPAccount_EmptyCurrencyOmitsDefaultAsset(t *testing.T) {
	got, err := SettingsAccountToPSPAccount(client.Account{ID: "acc_x", CurrencyCode: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.DefaultAsset != nil {
		t.Errorf("empty currency must yield nil DefaultAsset, got %v", *got.DefaultAsset)
	}
}

// AccountToBalance: amount converted to minor units, asset formatted,
// CreatedAt comes from the now param.
func TestAccountToBalance(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	in := client.Account{
		ID:           "acc_1",
		CurrencyCode: "USD",
		TypeDetails:  client.AccountTypeDetails{AvailableAmount: "1234.50"},
	}
	got, err := AccountToBalance(in, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AccountReference != "acc_1" {
		t.Errorf("AccountReference = %q, want acc_1", got.AccountReference)
	}
	if got.Asset != "USD/2" {
		t.Errorf("Asset = %q, want USD/2", got.Asset)
	}
	if got.Amount.Cmp(big.NewInt(123450)) != 0 {
		t.Errorf("Amount = %s, want 123450", got.Amount)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt mismatch")
	}
}

// USD fallback when CurrencyCode is empty (matches Routable's historical
// default behaviour).
func TestAccountToBalance_EmptyCurrencyDefaultsToUSD(t *testing.T) {
	got, err := AccountToBalance(client.Account{
		ID:          "acc_1",
		TypeDetails: client.AccountTypeDetails{AvailableAmount: "10.00"},
	}, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Asset != "USD/2" {
		t.Errorf("empty currency must default to USD/2, got %q", got.Asset)
	}
}

// Unsupported currency surfaces as an error so the caller can skip the row
// rather than silently emit a wrong-precision balance.
func TestAccountToBalance_UnknownCurrencyErrors(t *testing.T) {
	_, err := AccountToBalance(client.Account{
		ID:           "acc_x",
		CurrencyCode: "XYZ",
		TypeDetails:  client.AccountTypeDetails{AvailableAmount: "1.00"},
	}, time.Now())
	if err == nil {
		t.Fatal("expected error for unknown currency")
	}
}

// Bad amount string surfaces as an error (defensive — Routable returns
// well-shaped decimals normally, but we don't trust upstream).
func TestAccountToBalance_BadAmountErrors(t *testing.T) {
	_, err := AccountToBalance(client.Account{
		ID:           "acc_x",
		CurrencyCode: "USD",
		TypeDetails:  client.AccountTypeDetails{AvailableAmount: "not-a-number"},
	}, time.Now())
	if err == nil {
		t.Fatal("expected error for unparseable amount")
	}
}

// CompanyToPSPAccount: display_name preferred for Name; falls back to
// business_name when display_name is empty.
func TestCompanyToPSPAccount(t *testing.T) {
	created := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		company  client.Company
		wantName string
	}{
		{"display_name preferred", client.Company{ID: "co_1", DisplayName: "Acme Inc", BusinessName: "Acme Incorporated", CreatedAt: created}, "Acme Inc"},
		{"falls back to business_name", client.Company{ID: "co_2", BusinessName: "Inc.", CreatedAt: created}, "Inc."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CompanyToPSPAccount(tc.company)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name == nil || *got.Name != tc.wantName {
				t.Errorf("Name = %v, want *%s", got.Name, tc.wantName)
			}
		})
	}
}

// Both display_name and business_name absent → Name is nil (no empty
// strings, the Formance contract is `*string` precisely so callers can
// distinguish absent from set).
func TestCompanyToPSPAccount_BothNamesAbsent_NameNil(t *testing.T) {
	got, err := CompanyToPSPAccount(client.Company{ID: "co_3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != nil {
		t.Errorf("Name should be nil when neither display_name nor business_name set, got %v", *got.Name)
	}
}

// PayableToPSPPayment: full mapping with both source and destination set,
// metadata aliases populated.
func TestPayableToPSPPayment_FullShape(t *testing.T) {
	when := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	in := client.Payable{
		ID:                  "pa_1",
		Type:                "ach",
		DeliveryMethod:      "ach_standard",
		Status:              "completed",
		Amount:              "100.50",
		CurrencyCode:        "USD",
		ExternalID:          "pi_42",
		Memo:                "rent",
		Reference:           "ref-1",
		CreatedAt:           when,
		StatusChangedAt:     &when,
		PayToCompany:        &client.PayableCompany{ID: "co_to"},
		WithdrawFromAccount: &client.PayableAccount{ID: "acc_from"},
	}

	got, err := PayableToPSPPayment(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Reference != "pa_1" || got.Type != models.PAYMENT_TYPE_PAYOUT {
		t.Errorf("Reference/Type wrong: %+v", got)
	}
	if got.Status != models.PAYMENT_STATUS_SUCCEEDED {
		t.Errorf("Status = %v, want SUCCEEDED", got.Status)
	}
	if got.Scheme != models.PAYMENT_SCHEME_ACH {
		t.Errorf("Scheme = %v, want ACH", got.Scheme)
	}
	if got.Asset != "USD/2" {
		t.Errorf("Asset = %q, want USD/2", got.Asset)
	}
	if got.Amount == nil || got.Amount.Cmp(big.NewInt(10050)) != 0 {
		t.Errorf("Amount mismatch: %v", got.Amount)
	}
	if got.SourceAccountReference == nil || *got.SourceAccountReference != "acc_from" {
		t.Errorf("SourceAccountReference = %v", got.SourceAccountReference)
	}
	if got.DestinationAccountReference == nil || *got.DestinationAccountReference != "co_to" {
		t.Errorf("DestinationAccountReference = %v", got.DestinationAccountReference)
	}
	// Alias keys (the production correlation contract).
	if got.Metadata[MetadataKeyRoutablePayableID] != "pa_1" {
		t.Errorf("payable_id alias missing: %v", got.Metadata)
	}
	if got.Metadata[MetadataKeyPaymentInitiationReference] != "pi_42" {
		t.Errorf("payment_initiation_reference alias missing: %v", got.Metadata)
	}
}

// Invariant: an unsupported currency on a payable surfaces as an error
// (skip-and-log path in the parent fetcher), never a silent wrong-
// precision Payment.
func TestPayableToPSPPayment_UnknownCurrencyErrors(t *testing.T) {
	_, err := PayableToPSPPayment(client.Payable{ID: "pa_x", Status: "pending", Amount: "1.00", CurrencyCode: "XYZ"})
	if err == nil {
		t.Fatal("expected error for unknown currency")
	}
}

// Bad amount string also errors out cleanly.
func TestPayableToPSPPayment_BadAmountErrors(t *testing.T) {
	_, err := PayableToPSPPayment(client.Payable{ID: "pa_x", Status: "pending", Amount: "abc", CurrencyCode: "USD"})
	if err == nil {
		t.Fatal("expected error for unparseable amount")
	}
}

// CreatedAt feeds adjustment timestamps; prefer status_changed_at when
// present, fall back to created_at (nil on draft rows).
func TestPayableToPSPPayment_PrefersStatusChangedAt(t *testing.T) {
	created := time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)
	changed := time.Date(2025, 6, 1, 9, 30, 0, 0, time.UTC)

	withChanged, err := PayableToPSPPayment(client.Payable{
		ID: "pa_a", Status: "completed", Amount: "1.00", CurrencyCode: "USD",
		CreatedAt: created, StatusChangedAt: &changed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !withChanged.CreatedAt.Equal(changed) {
		t.Errorf("CreatedAt = %v, want StatusChangedAt %v", withChanged.CreatedAt, changed)
	}

	withoutChanged, err := PayableToPSPPayment(client.Payable{
		ID: "pa_b", Status: "draft", Amount: "1.00", CurrencyCode: "USD",
		CreatedAt: created, StatusChangedAt: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !withoutChanged.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want CreatedAt fallback %v", withoutChanged.CreatedAt, created)
	}

	zero := time.Time{}
	withZero, err := PayableToPSPPayment(client.Payable{
		ID: "pa_c", Status: "draft", Amount: "1.00", CurrencyCode: "USD",
		CreatedAt: created, StatusChangedAt: &zero,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !withZero.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want CreatedAt fallback %v (zero StatusChangedAt)", withZero.CreatedAt, created)
	}
}

// PayablesToPSPPayments: skips bad rows via the skip callback (so the
// caller can log + count) and tracks the latest StatusChangedAt observed.
func TestPayablesToPSPPayments_SkipsBadRowsAndTracksWatermark(t *testing.T) {
	t1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC)
	in := []client.Payable{
		{ID: "ok1", Status: "pending", Amount: "1.00", CurrencyCode: "USD", CreatedAt: t1, StatusChangedAt: &t1},
		{ID: "bad", Status: "pending", Amount: "1.00", CurrencyCode: "XYZ", CreatedAt: t2}, // unknown currency
		{ID: "ok2", Status: "pending", Amount: "1.00", CurrencyCode: "USD", CreatedAt: t2, StatusChangedAt: &t2},
	}
	skipped := []string{}
	out, watermark := PayablesToPSPPayments(in, time.Time{}, func(id string, _ error) { skipped = append(skipped, id) })

	if len(out) != 2 {
		t.Errorf("expected 2 mapped, got %d", len(out))
	}
	if len(skipped) != 1 || skipped[0] != "bad" {
		t.Errorf("expected skip(bad), got %v", skipped)
	}
	if !watermark.Equal(t2) {
		t.Errorf("watermark = %v, want %v", watermark, t2)
	}
}

// ReceivableToPSPPayment: mirror of payable mapping with PAYIN type and
// source/destination flipped.
func TestReceivableToPSPPayment(t *testing.T) {
	when := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	got, err := ReceivableToPSPPayment(client.Receivable{
		ID:               "re_1",
		Type:             "ach",
		Status:           "pending",
		Amount:           "5.00",
		CurrencyCode:     "USD",
		DeliveryMethod:   "ach_standard",
		ExternalID:       "pi_inbound",
		CreatedAt:        when,
		StatusChangedAt:  &when,
		PayFromCompany:   &client.ReceivableCompany{ID: "co_from"},
		DepositToAccount: &client.ReceivableAccount{ID: "acc_to"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Type != models.PAYMENT_TYPE_PAYIN {
		t.Errorf("Type = %v, want PAYIN", got.Type)
	}
	if got.SourceAccountReference == nil || *got.SourceAccountReference != "co_from" {
		t.Errorf("SourceAccountReference = %v", got.SourceAccountReference)
	}
	if got.DestinationAccountReference == nil || *got.DestinationAccountReference != "acc_to" {
		t.Errorf("DestinationAccountReference = %v", got.DestinationAccountReference)
	}
	if got.Metadata[MetadataKeyPaymentInitiationReference] != "pi_inbound" {
		t.Errorf("alias missing: %v", got.Metadata)
	}
}

// LaterOf: handles all four zero/non-zero combinations.
func TestLaterOf(t *testing.T) {
	zero := time.Time{}
	a := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		x, y time.Time
		want time.Time
	}{
		{"both zero", zero, zero, zero},
		{"a zero, b set", zero, b, b},
		{"a set, b zero", a, zero, a},
		{"a < b", a, b, b},
		{"a > b", b, a, b},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LaterOf(tc.x, tc.y); !got.Equal(tc.want) {
				t.Errorf("LaterOf = %v, want %v", got, tc.want)
			}
		})
	}
}

// StatusChangedAtOrCreated: prefers status_changed_at when non-nil/non-zero.
func TestStatusChangedAtOrCreated(t *testing.T) {
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	statusChanged := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	if got := StatusChangedAtOrCreated(&statusChanged, created); !got.Equal(statusChanged) {
		t.Errorf("preferred status_changed_at, got %v", got)
	}
	if got := StatusChangedAtOrCreated(nil, created); !got.Equal(created) {
		t.Errorf("nil status_changed_at must fall back to created_at, got %v", got)
	}
	zero := time.Time{}
	if got := StatusChangedAtOrCreated(&zero, created); !got.Equal(created) {
		t.Errorf("zero status_changed_at must fall back to created_at, got %v", got)
	}
}

// FormatAsset: known currency formats as ISO/precision (no slash for
// zero-precision currencies like JPY); unknown returns the input
// uppercase; empty returns empty. This mirrors go-libs/currency.FormatAsset.
func TestFormatAsset(t *testing.T) {
	cases := map[string]string{
		"USD": "USD/2",
		"usd": "USD/2", // case-insensitive
		"JPY": "JPY",   // precision 0 → no /N suffix
		"KWD": "KWD/3",
		"":    "",
		"XYZ": "XYZ", // unknown ISO code → returns input as-is (no /N)
	}
	for in, want := range cases {
		if got := FormatAsset(in); got != want {
			t.Errorf("FormatAsset(%q) = %q, want %q", in, got, want)
		}
	}
}

// SplitAsset: handles both prefixed ("USD/2") and bare ("USD") forms.
func TestSplitAsset(t *testing.T) {
	cases := []struct {
		in            string
		wantCurrency  string
		wantPrecision int
		wantErr       bool
	}{
		{"USD/2", "USD", 2, false},
		{"USD", "USD", 2, false},
		{"JPY/0", "JPY", 0, false},
		{"JPY", "JPY", 0, false},
		{"XYZ", "", 0, true},
		{"XYZ/2", "", 0, true}, // unknown currency, even with precision
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			c, p, err := SplitAsset(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if c != tc.wantCurrency || p != tc.wantPrecision {
				t.Errorf("SplitAsset(%q) = (%q, %d), want (%q, %d)", tc.in, c, p, tc.wantCurrency, tc.wantPrecision)
			}
		})
	}
}

// FieldOr: present non-empty wins; absent or empty falls back.
func TestFieldOr(t *testing.T) {
	m := map[string]string{"a": "x", "empty": ""}
	if got := FieldOr(m, "a", "fallback"); got != "x" {
		t.Errorf("FieldOr = %q, want x", got)
	}
	if got := FieldOr(m, "empty", "fallback"); got != "fallback" {
		t.Errorf("empty value must fall back, got %q", got)
	}
	if got := FieldOr(m, "missing", "fallback"); got != "fallback" {
		t.Errorf("missing key must fall back, got %q", got)
	}
}

// CompanyMetadata + SettingsAccountMetadata: stripEmpty drops zero-value
// fields so dashboards don't carry empty strings.
func TestCompanyMetadata_DropsEmptyFields(t *testing.T) {
	m := CompanyMetadata(client.Company{ID: "co_x", DisplayName: "Acme", IsVendor: true})
	if _, ok := m[MetadataPrefix+"display_name"]; !ok {
		t.Errorf("display_name should be present")
	}
	if _, ok := m[MetadataPrefix+"business_name"]; ok {
		t.Errorf("empty business_name should be stripped")
	}
	if m[MetadataPrefix+"is_vendor"] != "true" {
		t.Errorf("is_vendor must serialize as 'true'")
	}
	// boolString(false) returns "false" (not empty) — so the false case is
	// NOT stripped. We keep both true/false on disk so dashboards can
	// distinguish "explicitly false" from "absent".
	if m[MetadataPrefix+"is_archived"] != "false" {
		t.Errorf("is_archived false must serialize as 'false', got %q", m[MetadataPrefix+"is_archived"])
	}
}

func TestCompanyMetadata_AddressFlattenedAndStripped(t *testing.T) {
	m := CompanyMetadata(client.Company{
		ID:                "co_x",
		DisplayName:       "Acme",
		RegisteredAddress: &client.Address{Line1: "1 Main St", City: "NYC", Country: "US"},
	})
	if m[MetadataPrefix+"address.line_1"] != "1 Main St" {
		t.Errorf("address.line_1 missing or wrong: %v", m)
	}
	if m[MetadataPrefix+"address.city"] != "NYC" {
		t.Errorf("address.city missing or wrong: %v", m)
	}
	if _, ok := m[MetadataPrefix+"address.line_2"]; ok {
		t.Errorf("empty address.line_2 should be stripped")
	}
}

func TestSettingsAccountMetadata(t *testing.T) {
	m := SettingsAccountMetadata(client.Account{
		Object:       "Account",
		Type:         "checking",
		IsValid:      true,
		CurrencyCode: "USD",
		TypeDetails: client.AccountTypeDetails{
			AccountType:   "business",
			BankName:      "JPM",
			AccountNumber: "***1",
			RoutingNumber: "021",
		},
	})
	if m[MetadataPrefix+"currency_code"] != "USD" {
		t.Errorf("currency_code missing")
	}
	if m[MetadataPrefix+"is_valid"] != "true" {
		t.Errorf("is_valid must be 'true', got %q", m[MetadataPrefix+"is_valid"])
	}
	if m[MetadataPrefix+"bank_name"] != "JPM" {
		t.Errorf("bank_name missing")
	}
}
