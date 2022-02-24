package payment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/numary/go-libs/sharedapi"
	payment "github.com/numary/payment/pkg"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func runApiWithMock(t *testing.T, fn func(t *mtest.T, mux *mux.Router)) {
	runWithMock(t, func(t *mtest.T) {
		fn(t, payment.NewMux(payment.NewDefaultService(t.DB)))
	})
}

func TestHttpServerCreatePayment(t *testing.T) {
	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/organizations/foo/payments", bytes.NewBufferString(`{}`))

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)

		// TODO: Check result
	})
}

func TestHttpServerUpdatePayment(t *testing.T) {
	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
		_, err := t.DB.Collection("Payment").InsertOne(context.Background(), map[string]interface{}{
			"_id":            "1",
			"organizationId": "foo",
		})
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/organizations/foo/payments", bytes.NewBufferString(`{"id": "1"}`))

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
	})
}

func TestHttpServerListPayments(t *testing.T) {
	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
		_, err := t.DB.Collection("Payment").InsertMany(context.Background(), []interface{}{
			map[string]interface{}{
				"_id":            "1",
				"organizationId": "foo",
			},
			map[string]interface{}{
				"_id":            "2",
				"organizationId": "foo",
			},
			map[string]interface{}{
				"_id":            "3",
				"organizationId": "foo",
			},
		})
		assert.NoError(t, err)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/organizations/foo/payments", nil)
		values := url.Values{}
		values.Set("limit", "2")
		values.Set("sort", "id:desc")
		req.URL.RawQuery = values.Encode()

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)

		type Response struct {
			sharedapi.BaseResponse
			Data []payment.Payment `json:"data"`
		}
		ret := &Response{}
		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&ret))
		assert.Len(t, ret.Data, 2)
	})
}
