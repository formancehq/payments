package payment_test

import (
	"bytes"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"net/http"
	"net/http/httptest"
	payment "payment/pkg"
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
		t.AddMockResponses(mtest.CreateCursorResponse(0, t.Name()+".Payment", mtest.FirstBatch, bson.D{}))

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/organizations/foo/payments", nil)

		m.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
	})
}
