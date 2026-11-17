package v3

// Regression coverage for stored credentials in the v3 connector list.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/api/backend"
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

// sec92ListLeakedKeys parses a v3 list response and reports which sensitive
// config keys are present in any row. Only key names and value lengths are
// reported, never values.
func sec92ListLeakedKeys(body []byte) []string {
	var probe struct {
		Cursor struct {
			Data []struct {
				Config map[string]json.RawMessage `json:"config"`
			} `json:"data"`
		} `json:"cursor"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return []string{"unparseable-response"}
	}
	leaked := []string{}
	for _, row := range probe.Cursor.Data {
		for _, key := range []string{"apiKey", "apiSecret", "secret", "clientSecret", "password", "privateKey"} {
			if raw, ok := row.Config[key]; ok {
				leaked = append(leaked, key+"(len="+strconv.Itoa(len(raw))+")")
			}
		}
	}
	return leaked
}

func sec92ListBody(t *testing.T, m *backend.MockBackend, stored json.RawMessage) []byte {
	t.Helper()
	connectorID := models.ConnectorID{Reference: uuid.New(), Provider: "STRIPE"}
	m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
		&paginate.Cursor[models.Connector]{Data: []models.Connector{{
			ConnectorBase: models.ConnectorBase{ID: connectorID, Provider: "STRIPE", Name: "sec92-fixture"},
			Config:        stored,
		}}},
		nil,
	)
	m.EXPECT().ConnectorsCapabilities().Return(map[string][]models.Capability{})

	w := httptest.NewRecorder()
	connectorsList(m)(w, httptest.NewRequest(http.MethodGet, "/", nil))
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return body
}

// TestSEC92V3ConnectorsListMustNotLeakStoredSecret is the API-level RED:
// GET /v3/connectors must not expose stored PSP credentials in row configs.
func TestSEC92V3ConnectorsListMustNotLeakStoredSecret(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := backend.NewMockBackend(ctrl)

	body := sec92ListBody(t, m, sec92StoredConfig())
	if leaked := sec92ListLeakedKeys(body); len(leaked) > 0 {
		t.Fatalf("SEC-92 reproduced at API boundary: v3 connectors list exposes stored secret keys %v (values never printed)", leaked)
	}
}
