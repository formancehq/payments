package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Payment Service Users Connections List", func() {
	var (
		handlerFn   http.HandlerFunc
		psuID       uuid.UUID
		connectorID models.ConnectorID
	)

	BeforeEach(func() {
		psuID = uuid.New()
		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "test"}
	})

	Context("list all psu connections", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersConnectionsListAll(m)
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String())
			expectedErr := errors.New("psu connections list error")
			m.EXPECT().PaymentServiceUsersConnectionsList(gomock.Any(), psuID, nil, gomock.Any()).Return(
				&bunpaginate.Cursor[models.PSUBankBridgeConnection]{}, expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "paymentServiceUserID", psuID.String())
			cursor := &bunpaginate.Cursor[models.PSUBankBridgeConnection]{}
			m.EXPECT().PaymentServiceUsersConnectionsList(gomock.Any(), psuID, nil, gomock.Any()).Return(cursor, nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})

	Context("list psu connections from connector ID", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentServiceUsersConnectionsListFromConnectorID(m)
		})

		It("should return an invalid ID error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalidvalue", "paymentServiceUserID", psuID.String())
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an invalid ID error when psu ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connectorID.String(), "paymentServiceUserID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connectorID.String(), "paymentServiceUserID", psuID.String())
			expectedErr := errors.New("psu connections list error")
			m.EXPECT().PaymentServiceUsersConnectionsList(gomock.Any(), psuID, &connectorID, gomock.Any()).Return(
				&bunpaginate.Cursor[models.PSUBankBridgeConnection]{}, expectedErr,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connectorID.String(), "paymentServiceUserID", psuID.String())
			cursor := &bunpaginate.Cursor[models.PSUBankBridgeConnection]{}
			m.EXPECT().PaymentServiceUsersConnectionsList(gomock.Any(), psuID, &connectorID, gomock.Any()).Return(cursor, nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
