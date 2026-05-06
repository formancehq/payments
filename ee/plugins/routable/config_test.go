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
	if cfg.PollingPeriod.Duration() == 0 {
		t.Error("expected non-zero polling period default")
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
