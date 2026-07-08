package krakenpro

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalConfigAcceptsValid(t *testing.T) {
	t.Parallel()
	cfg, err := unmarshalAndValidateConfig(json.RawMessage(`{"apiKey":"k","apiSecret":"YWJjZA==","endpoint":"https://api.uat.kraken.com"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if cfg.APIKey != "k" || cfg.Endpoint != "https://api.uat.kraken.com" {
		t.Errorf("unexpected config: %+v", cfg)
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
