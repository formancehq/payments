package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Schedules List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list schedules", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = schedulesList(m)
		})

		It("should return an validation error when connector ID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, "INVALID")
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			connectorID := models.ConnectorID{Provider: "psp", Reference: uuid.New()}
			req := prepareQueryRequest(http.MethodGet, "connectorID", connectorID.String())
			m.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{}, fmt.Errorf("schedules list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			connectorID := models.ConnectorID{Provider: "psp", Reference: uuid.New()}
			req := prepareQueryRequest(http.MethodGet, "connectorID", connectorID.String())
			m.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
