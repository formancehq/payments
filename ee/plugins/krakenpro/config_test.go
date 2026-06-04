package krakenpro

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/sharedconfig"
)

func TestUnmarshalConfigAppliesDefaultPollingPeriod(t *testing.T) {
	t.Parallel()
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA==","endpoint":"https://api.uat.kraken.com"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if cfg.PollingPeriod.Duration() != sharedconfig.DefaultPollingPeriod {
		t.Errorf("polling=%v want default %v", cfg.PollingPeriod.Duration(), sharedconfig.DefaultPollingPeriod)
	}
}

func TestUnmarshalConfigEnforcesMinPollingPeriod(t *testing.T) {
	t.Parallel()
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA==","endpoint":"https://api.uat.kraken.com","pollingPeriod":"5m"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if cfg.PollingPeriod.Duration() != sharedconfig.MinimumPollingPeriod {
		t.Errorf("polling=%v want min %v", cfg.PollingPeriod.Duration(), sharedconfig.MinimumPollingPeriod)
	}
}

func TestUnmarshalConfigAcceptsExplicitPollingPeriod(t *testing.T) {
	t.Parallel()
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA==","endpoint":"https://api.uat.kraken.com","pollingPeriod":"1h"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if cfg.PollingPeriod.Duration() != time.Hour {
		t.Errorf("polling=%v want 1h", cfg.PollingPeriod.Duration())
	}
}

func TestUnmarshalConfigRejectsInvalidURL(t *testing.T) {
	t.Parallel()
	_, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA==","endpoint":"::not-a-url"}`))
	if err == nil {
		t.Fatal("expected URL validation error")
	}
}

func TestUnmarshalConfigRequiresEndpoint(t *testing.T) {
	t.Parallel()
	// The VIP dialect is incompatible with the public api.kraken.com
	// default, so a blank endpoint must fail fast rather than silently
	// fall back to the wrong host.
	_, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA=="}`))
	if err == nil {
		t.Fatal("expected endpoint-required validation error")
	}
}
