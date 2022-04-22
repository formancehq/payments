package payment

import (
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func forgeToken(t *testing.T) string {
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{}).SignedString([]byte("0000000000000000"))
	assert.NoError(t, err)
	return tok
}

func TestAuthUnauthorized(t *testing.T) {

	m := AuthMiddleware()
	r := mux.NewRouter()
	r.Path("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Use(m)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
}

func TestHttpBasic(t *testing.T) {

	m := AuthMiddleware(NewHTTPBasicMethod(map[string]string{
		"foo": "bar",
	}))
	r := mux.NewRouter()
	r.Path("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Use(m)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("foo", "bar")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
}

func TestHttpBasicForbidden(t *testing.T) {

	m := AuthMiddleware(NewHTTPBasicMethod(map[string]string{
		"foo": "bar",
	}))
	r := mux.NewRouter()
	r.Path("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Use(m)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("foo", "baz")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Result().StatusCode)
}

func TestHttpBearer(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := AuthMiddleware(NewHttpBearerMethod(http.DefaultClient, srv.URL))
	r := mux.NewRouter()
	r.Path("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Use(m)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+forgeToken(t))

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
}

func TestHttpBearerForbidden(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	m := AuthMiddleware(NewHttpBearerMethod(http.DefaultClient, srv.URL))
	r := mux.NewRouter()
	r.Path("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Use(m)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer XXX")

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Result().StatusCode)
}
