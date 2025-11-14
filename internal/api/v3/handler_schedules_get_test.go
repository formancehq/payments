package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 Schedules Get", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("get a schedule", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = schedulesGet(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connID.String())
			m.EXPECT().SchedulesGet(gomock.Any(), gomock.Any(), connID).Return(
				nil, fmt.Errorf("schedules list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return bad request error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, "INVALID_ID")
		})

		It("should return data object", func(ctx SpecContext) {
			scheduleID := "someID"
			req := prepareQueryRequest(http.MethodGet, "connectorID", connID.String(), "scheduleID", scheduleID)
			m.EXPECT().SchedulesGet(gomock.Any(), scheduleID, connID).Return(
				&models.Schedule{
					ID:          scheduleID,
					ConnectorID: connID,
					CreatedAt:   time.Now(),
				}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
