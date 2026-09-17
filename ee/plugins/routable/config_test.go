package routable

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.resolvedEndpoint() != "https://api.routable.com" {
		t.Errorf("expected default endpoint, got %q", cfg.resolvedEndpoint())
	}
	if cfg.ActingTeamMember != "" {
		t.Errorf("expected empty acting team member, got %q", cfg.ActingTeamMember)
	}
}

func TestConfigRejectsMissingAPIKey(t *testing.T) {
	if _, err := unmarshalAndValidateConfig(json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected validation error on missing apiKey")
	}
}

func TestConfigRejectsBadEndpoint(t *testing.T) {
	_, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","endpoint":"not-a-url"}`))
	if err == nil {
		t.Fatal("expected validation error on non-URL endpoint")
	}
}

func TestConfigLeavesPayoutsPerMinuteUnsetWhenAbsent(t *testing.T) {
	// A connector installed before the field existed has no such key in its
	// stored config. It must read as unset so the plugin applies its own
	// default rather than a rate the caller never asked for.
	for _, payload := range []string{
		`{"apiKey":"k"}`,
		`{"apiKey":"k","payoutsPerMinute":null}`,
	} {
		cfg, err := unmarshalAndValidateConfig(json.RawMessage(payload))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", payload, err)
		}
		if cfg.PayoutsPerMinute != nil {
			t.Errorf("%s: expected unset payouts/min, got %d", payload, *cfg.PayoutsPerMinute)
		}
	}
}

func TestConfigAcceptsCustomPayoutsPerMinute(t *testing.T) {
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","payoutsPerMinute":255}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PayoutsPerMinute == nil || *cfg.PayoutsPerMinute != 255 {
		t.Errorf("expected 255 payouts/min, got %v", cfg.PayoutsPerMinute)
	}
}

func TestConfigRejectsZeroPayoutsPerMinute(t *testing.T) {
	if _, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","payoutsPerMinute":0}`)); err == nil {
		t.Fatal("expected validation error on a zero payoutsPerMinute")
	}
}

func TestConfigRoundTripsUnsetPayoutsPerMinute(t *testing.T) {
	// The stored config is re-marshalled from this struct (combineConfigs) and
	// read back on every load. An unset rate must not come back as an explicit
	// 0, which would now be rejected.
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(string(b), `"payoutsPerMinute":0`) {
		t.Errorf("expected an unset rate not to marshal as 0, got %s", b)
	}
	reread, err := unmarshalAndValidateConfig(b)
	if err != nil {
		t.Fatalf("re-reading the marshalled config failed: %v", err)
	}
	if reread.PayoutsPerMinute != nil {
		t.Errorf("expected the rate to stay unset, got %d", *reread.PayoutsPerMinute)
	}
}

func TestConfigRejectsNegativeOrFractionalPayoutsPerMinute(t *testing.T) {
	// The unsigned, whole type is what refuses these, at unmarshal: a negative
	// rate is a caller mistake, and truncating 90.5 to 90 would hide one.
	for _, payload := range []string{
		`{"apiKey":"k","payoutsPerMinute":-30}`,
		`{"apiKey":"k","payoutsPerMinute":90.5}`,
	} {
		if _, err := unmarshalAndValidateConfig(json.RawMessage(payload)); err == nil {
			t.Errorf("%s: expected an error", payload)
		}
	}
}
