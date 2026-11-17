package services

// SEC-92 API/service-boundary reproduction (test-only, synthetic fixtures).
//
// Both the v3 GET /v3/connectors/{connectorID}/config handler and the legacy
// v1/v2 GET /connectors/{connector}/{id}/config handler call
// Service.ConnectorsConfig, which unmarshals the whole stored (decrypted)
// config blob, injects only a "provider" discriminator, and returns it
// verbatim. That is the direct service boundary for the secret leak.
//
// Assertions are boolean/key-absence only: no secret value (even a synthetic
// one) is ever written to test output.

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// sec92StoredConfig is a synthetic stored config blob shaped like a real PSP
// config (stripe: name + apiKey + pollingPeriod). The apiKey value is a
// synthetic marker, never a real credential; it is only used to detect key
// presence and its value is never printed.
func sec92StoredConfig() json.RawMessage {
	const marker = "sec92-synthetic-marker-not-a-secret"
	return json.RawMessage(`{"name":"sec92-fixture","apiKey":"` + marker + `","pollingPeriod":"30m"}`)
}

// sec92LeakedKeys reports which sensitive keys survive in the returned config.
// Only key names and value lengths are reported, never values.
func sec92LeakedKeys(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return []string{"unparseable-response"}
	}
	leaked := []string{}
	for _, key := range []string{"apiKey", "apiSecret", "secret", "clientSecret", "password", "privateKey", "passphrase", "webhookPassword", "webhookSharedSecret", "userCertificate", "userCertificateKey", "configurationToken", "accessKey"} {
		if v, ok := m[key]; ok {
			leaked = append(leaked, key+"(len="+strconv.Itoa(len(v))+")")
		}
	}
	return leaked
}

// TestSEC92ConnectorsConfigMustNotLeakStoredSecret is the service-level RED:
// ConnectorsConfig (backing GET config on v1/v2/v3) must not return stored
// PSP credentials.
func TestSEC92ConnectorsConfigMustNotLeakStoredSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)

	s := New(store, eng, false)

	connectorID := models.ConnectorID{Provider: "stripe", Reference: uuid.New()}
	store.EXPECT().ConnectorsGet(gomock.Any(), connectorID).Return(&models.Connector{
		ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "stripe", Name: "sec92-fixture"},
		Config:        sec92StoredConfig(),
	}, nil)

	raw, err := s.ConnectorsConfig(t.Context(), connectorID)
	if err != nil {
		t.Fatalf("ConnectorsConfig: %v", err)
	}
	if leaked := sec92LeakedKeys(t, raw); len(leaked) > 0 {
		t.Fatalf("SEC-92 reproduced at service boundary: ConnectorsConfig returns stored secret keys %v (values never printed)", leaked)
	}

	// The provider discriminator must still be injected (contract preserved).
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["provider"]; !ok {
		t.Fatalf("provider discriminator missing from response")
	}
}
