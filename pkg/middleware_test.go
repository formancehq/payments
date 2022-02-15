package payment_test

import (
	"bytes"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs-cloud/pkg/auth"
	"github.com/numary/go-libs-cloud/pkg/middlewares"
	payment "github.com/numary/payment/pkg"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	runApiWithMock(t, func(t *mtest.T, mux *mux.Router) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		mux = payment.ConfigureAuthMiddleware(mux, middlewares.AuthMiddleware(testServer.URL))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/organizations/foo/payments", bytes.NewBufferString("{}"))
		req.Header.Set("Authorization", "Bearer XXX")

		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	})
}

func TestCheckOrganizationAccessMiddleware(t *testing.T) {
	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		m = payment.ConfigureAuthMiddleware(m, middlewares.CheckOrganizationAccessMiddleware(func(r *http.Request, name string) string {
			return mux.Vars(r)[name]
		}))

		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.ClaimStruct{
			Organizations: []auth.ClaimOrganization{{Name: "foo"}},
		}).SignedString([]byte("0000000000000000"))
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/organizations/foo/payments", bytes.NewBufferString("{}"))
		req.Header.Set("Authorization", "Bearer "+token)

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	})
}
