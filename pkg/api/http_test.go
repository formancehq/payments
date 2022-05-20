package api_test

//
//import (
//	"bytes"
//	"context"
//	"encoding/json"
//	"github.com/gorilla/mux"
//	"github.com/numary/go-libs/sharedapi"
//	payment "github.com/numary/payments/pkg"
//	http2 "github.com/numary/payments/pkg/http"
//	"github.com/numary/payments/pkg/ingester"
//	testing2 "github.com/numary/payments/pkg/testing"
//	"github.com/stretchr/testify/assert"
//	"go.mongodb.org/mongo-driver/bson"
//	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
//	"go.mongodb.org/mongo-driver/mongo/options"
//	"net/http"
//	"net/http/httptest"
//	"net/url"
//	"testing"
//	"time"
//)
//
//func runApiWithMock(t *testing.T, fn func(t *mtest.T, mux *mux.Router)) {
//	testing2.RunWithMock(t, func(t *mtest.T) {
//		fn(t, http2.NewMux(ingester.NewDefaultService(t.DB), false))
//	})
//}
//
//func TestHttpServerCreatePayment(t *testing.T) {
//	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
//		rec := httptest.NewRecorder()
//		req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{}`))
//
//		m.ServeHTTP(rec, req)
//
//		assert.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
//	})
//}
//
//func TestHttpServerUpdatePayment(t *testing.T) {
//	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
//		_, err := t.DB.Collection(ingester.Collection).InsertOne(context.Background(), map[string]interface{}{
//			"id":   "1",
//			"date": time.Now(),
//		})
//		assert.NoError(t, err)
//
//		rec := httptest.NewRecorder()
//		req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"id": "1", "scheme": "visa", "date": "`+time.Now().Add(time.Minute).Format(time.RFC3339)+`"}`))
//
//		m.ServeHTTP(rec, req)
//
//		assert.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
//
//		ret := t.DB.Collection(ingester.Collection).FindOne(context.Background(), map[string]interface{}{
//			"id": "1",
//		}, options.FindOne().SetSort(bson.M{"date": -1}))
//		assert.NoError(t, ret.Err())
//
//		p := payment.Payment{}
//		assert.NoError(t, ret.Decode(&p))
//		assert.Equal(t, "visa", p.Scheme)
//	})
//}
//
//func TestHttpServerListPayments(t *testing.T) {
//	runApiWithMock(t, func(t *mtest.T, m *mux.Router) {
//		_, err := t.DB.Collection("Payment").InsertMany(context.Background(), []interface{}{
//			map[string]interface{}{
//				"id": "1",
//			},
//			map[string]interface{}{
//				"id": "2",
//			},
//			map[string]interface{}{
//				"id": "3",
//			},
//		})
//		assert.NoError(t, err)
//
//		rec := httptest.NewRecorder()
//		req := httptest.NewRequest(http.MethodGet, "/", nil)
//		values := url.Values{}
//		values.Set("limit", "2")
//		values.Set("sort", "id:desc")
//		req.URL.RawQuery = values.Encode()
//
//		m.ServeHTTP(rec, req)
//
//		assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
//
//		type Response struct {
//			sharedapi.BaseResponse
//			Data []payment.Payment `json:"data"`
//		}
//		ret := &Response{}
//		assert.NoError(t, json.NewDecoder(rec.Body).Decode(&ret))
//		assert.Len(t, ret.Data, 2)
//	})
//}
