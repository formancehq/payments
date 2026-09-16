package routable

import (
	"encoding/json"
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

func TestConfigLeavesPayoutsPerMinuteAtZeroWhenAbsent(t *testing.T) {
	// A connector installed before the field existed has no such key in its
	// stored config. It must read as 0 so the plugin applies its own default.
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PayoutsPerMinute != 0 {
		t.Errorf("expected 0 payouts/min, got %d", cfg.PayoutsPerMinute)
	}
}

func TestConfigAcceptsCustomPayoutsPerMinute(t *testing.T) {
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","payoutsPerMinute":255}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PayoutsPerMinute != 255 {
		t.Errorf("expected 255 payouts/min, got %d", cfg.PayoutsPerMinute)
	}
}

func TestConfigAcceptsZeroAndNullPayoutsPerMinute(t *testing.T) {
	// omitempty has to let a zero through: 0 is the unset marker, not a
	// rejected value. Were it rejected, every connector stored before the
	// field existed would fail to load.
	for _, payload := range []string{
		`{"apiKey":"k","payoutsPerMinute":0}`,
		`{"apiKey":"k","payoutsPerMinute":null}`,
	} {
		cfg, err := unmarshalAndValidateConfig(json.RawMessage(payload))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", payload, err)
		}
		if cfg.PayoutsPerMinute != 0 {
			t.Errorf("%s: expected 0 payouts/min, got %d", payload, cfg.PayoutsPerMinute)
		}
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
