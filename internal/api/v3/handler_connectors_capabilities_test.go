package v3

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/services"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Connectors Capabilities", func() {
	Context("list capabilities catalog", func() {
		var (
			w         *httptest.ResponseRecorder
			m         *backend.MockBackend
			handlerFn http.HandlerFunc
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsCapabilities(m)
		})

		It("returns the catalog with ETag and Cache-Control headers", func(ctx SpecContext) {
			m.EXPECT().ConnectorsCapabilities().Return(map[string][]models.Capability{
				"stripe": {models.CAPABILITY_FETCH_ACCOUNTS, models.CAPABILITY_CREATE_TRANSFER},
			})

			handlerFn(w, httptest.NewRequest(http.MethodGet, "/", nil))

			res := w.Result()
			defer res.Body.Close()
			Expect(res.StatusCode).To(Equal(http.StatusOK))
			Expect(res.Header.Get("Cache-Control")).To(Equal(capabilitiesCacheControl))
			etag := res.Header.Get("ETag")
			Expect(etag).To(HavePrefix(`"`))
			Expect(etag).To(HaveSuffix(`"`))

			var body struct {
				Data map[string][]models.Capability `json:"data"`
			}
			Expect(json.NewDecoder(res.Body).Decode(&body)).To(Succeed())
			Expect(body.Data).To(HaveKeyWithValue("stripe", []models.Capability{
				models.CAPABILITY_FETCH_ACCOUNTS,
				models.CAPABILITY_CREATE_TRANSFER,
			}))
		})

		It("returns 304 when If-None-Match matches the cached ETag", func(ctx SpecContext) {
			// OnceValues caches per-handler so the second request hits the
			// same ETag without touching the backend again.
			m.EXPECT().ConnectorsCapabilities().Return(map[string][]models.Capability{
				"stripe": {models.CAPABILITY_FETCH_ACCOUNTS},
			}).Times(1)

			first := httptest.NewRecorder()
			handlerFn(first, httptest.NewRequest(http.MethodGet, "/", nil))
			etag := first.Result().Header.Get("ETag")

			second := httptest.NewRequest(http.MethodGet, "/", nil)
			second.Header.Set("If-None-Match", etag)
			handlerFn(w, second)

			res := w.Result()
			defer res.Body.Close()
			Expect(res.StatusCode).To(Equal(http.StatusNotModified))
			Expect(w.Body.Len()).To(Equal(0))
		})
	})

	Context("get capabilities for a single connector", func() {
		var (
			w         *httptest.ResponseRecorder
			m         *backend.MockBackend
			handlerFn http.HandlerFunc
			connID    models.ConnectorID
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsCapabilitiesGet(m)
			connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		})

		It("returns 400 when the connector ID is malformed", func(ctx SpecContext) {
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", "not-an-id"))
			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("returns 404 when the connector is not found", func(ctx SpecContext) {
			m.EXPECT().ConnectorsCapabilitiesGet(gomock.Any(), connID).
				Return(nil, services.ErrNotFound)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusNotFound, "")
		})

		It("returns 404 when the underlying storage cannot find the row", func(ctx SpecContext) {
			m.EXPECT().ConnectorsCapabilitiesGet(gomock.Any(), connID).
				Return(nil, storage.ErrNotFound)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusNotFound, "")
		})

		It("returns 500 for unexpected errors", func(ctx SpecContext) {
			m.EXPECT().ConnectorsCapabilitiesGet(gomock.Any(), connID).
				Return(nil, errors.New("boom"))
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("returns capability names on success", func(ctx SpecContext) {
			caps := []models.Capability{
				models.CAPABILITY_FETCH_ACCOUNTS,
				models.CAPABILITY_CREATE_TRANSFER,
			}
			m.EXPECT().ConnectorsCapabilitiesGet(gomock.Any(), connID).Return(caps, nil)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "connectorID", connID.String()))

			res := w.Result()
			defer res.Body.Close()
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			var body struct {
				Data []models.Capability `json:"data"`
			}
			Expect(json.NewDecoder(res.Body).Decode(&body)).To(Succeed())
			Expect(body.Data).To(Equal(caps))
		})
	})
})
