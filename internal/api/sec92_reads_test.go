package api_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	"github.com/formancehq/go-libs/v5/pkg/authn/oidc"
	"github.com/formancehq/go-libs/v5/pkg/service/health"
	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	transportapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
	libtime "github.com/formancehq/go-libs/v5/pkg/types/time"
	paymentsapi "github.com/formancehq/payments/internal/api"
	"github.com/formancehq/payments/internal/api/services"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/mock/gomock"
)

// Exercise actual routers, JWT verification, service redaction, and response
// encoding. Keys and stored credentials are generated/synthetic local fixtures;
// neither issuer discovery nor a PSP is contacted. The legacy v1/v2 contract
// shares the unprefixed v2 router in this service; upstream prefix rewriting is
// outside this test.
func TestSEC92StoredCredentialsExcludedFromHTTPReads(t *testing.T) {
	var versions struct {
		fx.In
		Values []paymentsapi.Version `group:"apiVersions"`
	}
	app := fx.New(fx.NopLogger, v2.NewModule(), v3.NewModule(), fx.Populate(&versions))
	require.NoError(t, app.Err())
	require.Len(t, versions.Values, 2)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const issuer = "https://sec92-fixture.invalid"
	keySet := oidc.NewStaticKeySet(jose.JSONWebKey{Key: &key.PublicKey, KeyID: "local", Algorithm: string(jose.RS256), Use: oidc.KeyUseSignature})
	auth := jwt.NewJWTAuth(map[string]oidc.KeySet{issuer: keySet}, "payments", true, nil)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "local"))
	require.NoError(t, err)

	tokens := make(map[string]string)
	for _, scope := range []string{"payments:read", "payments:read webhooks:read"} {
		claims := oidc.NewAccessTokenClaims(issuer, "fixture", []string{"payments"}, libtime.New(time.Now().Add(time.Hour)), uuid.NewString(), "fixture")
		claims.Scopes = strings.Fields(scope)
		payload, err := json.Marshal(claims)
		require.NoError(t, err)
		signed, err := signer.Sign(payload)
		require.NoError(t, err)
		tokens[scope], err = signed.CompactSerialize()
		require.NoError(t, err)
	}

	for _, provider := range []string{"STRIPE", "unregistered-provider"} {
		id := models.ConnectorID{Provider: provider, Reference: uuid.New()}
		// Unknown nested fields are deliberately unrelated to known credential names.
		stored := json.RawMessage(`{"name":"fixture","pollingPeriod":"30m","apiKey":"fake","opaque":{"futureField":"fake-nested"},"clientSecret":"fake"}`)
		connector := models.Connector{ConnectorBase: models.ConnectorBase{ID: id, Provider: provider, Name: "fixture"}, Config: stored}
		for scope, token := range tokens {
			for _, path := range []string{"/connectors/" + provider + "/" + id.String() + "/config", "/v3/connectors/" + id.String() + "/config", "/v3/connectors/"} {
				t.Run(provider+"/"+scope+path, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					store := storage.NewMockStorage(ctrl)
					eng := engine.NewMockEngine(ctrl)
					if strings.HasSuffix(path, "/config") {
						store.EXPECT().ConnectorsGet(gomock.Any(), id).Return(&connector, nil)
					} else {
						store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&paginate.Cursor[models.Connector]{Data: []models.Connector{connector}}, nil)
					}
					svc := services.New(store, eng, false)
					router := paymentsapi.NewRouter(svc, transportapi.ServiceInfo{}, &health.HealthController{}, auth, nil, false, versions.Values...)
					req := httptest.NewRequest(http.MethodGet, path, nil)
					req.Header.Set("Authorization", "Bearer "+token)
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, req)
					require.Equal(t, http.StatusOK, rec.Code)
					var envelope struct {
						Data   map[string]json.RawMessage `json:"data"`
						Cursor struct {
							Data []struct {
								Config map[string]json.RawMessage `json:"config"`
							} `json:"data"`
						} `json:"cursor"`
					}
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
					fields := envelope.Data
					if !strings.HasSuffix(path, "/config") {
						require.Len(t, envelope.Cursor.Data, 1, "a nonempty list must be tested")
						fields = envelope.Cursor.Data[0].Config
					}
					require.NotEmpty(t, fields)
					require.Contains(t, fields, "name")
					require.Contains(t, fields, "pollingPeriod")
					for name := range fields {
						require.Contains(t, []string{"name", "pollingPeriod", "provider"}, name, "response includes a provider field")
					}
					// Compare bytes only as a boolean to keep fixture credentials out of logs.
					require.False(t, strings.Contains(rec.Body.String(), "fake"), "stored credential marker reached response")
				})
			}
		}
	}

	t.Run("unauthenticated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		svc := services.New(storage.NewMockStorage(ctrl), engine.NewMockEngine(ctrl), false)
		router := paymentsapi.NewRouter(svc, transportapi.ServiceInfo{}, &health.HealthController{}, auth, nil, false, versions.Values...)
		for _, path := range []string{"/connectors/stripe/fixture/config", "/v3/connectors/fixture/config", "/v3/connectors/"} {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		}
	})
}
