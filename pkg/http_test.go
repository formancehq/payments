package payment_test

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	payment "github.com/numary/payment/pkg"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func runApiWithMock(t *testing.T, name string, fn func(t *mtest.T, mux *mux.Router)) {
	runWithMock(t, name, func(t *mtest.T) {
		fn(t, payment.NewMux(payment.NewDefaultService(t.DB)))
	})
}

func TestHttpServerCreatePayment(t *testing.T) {
	runApiWithMock(t, "CreatePayment", func(t *mtest.T, m *mux.Router) {
		t.AddMockResponses(mtest.CreateSuccessResponse())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/organizations/foo/payments", bytes.NewBufferString(`{}`))

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	})
}

func TestHttpServerUpdatePayment(t *testing.T) {
	runApiWithMock(t, "UpdatePayment", func(t *mtest.T, m *mux.Router) {
		t.AddMockResponses(mtest.CreateSuccessResponse())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/organizations/foo/payments/1", bytes.NewBufferString(`{}`))

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
	})
}

func TestHttpServerListPayments(t *testing.T) {
	runApiWithMock(t, "UpdatePayment", func(t *mtest.T, m *mux.Router) {
		t.AddMockResponses(mtest.CreateCursorResponse(0, t.Name()+".Payment", mtest.FirstBatch, bson.D{
			{
				Key:   "_id",
				Value: uuid.New(),
			},
		}, bson.D{
			{
				Key:   "_id",
				Value: uuid.New(),
			},
		}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/organizations/foo/payments", nil)
		values := url.Values{}
		values.Set("limit", "2")
		values.Set("sort", "id:desc")
		req.URL.RawQuery = values.Encode()

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
		ret := make([]payment.Payment, 0)
		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&ret))
		assert.Len(t, ret, 2)
	})
}
