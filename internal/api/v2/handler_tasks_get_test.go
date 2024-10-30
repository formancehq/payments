package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Get Task", func() {
	var (
		handlerFn http.HandlerFunc
		taskID    models.TaskID
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		taskID = models.TaskID{Reference: "ref", ConnectorID: connID}
	})

	Context("get tasks", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = tasksGet(m)
		})

		It("should return an invalid ID error when connectorID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest("connectorID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest("connectorID", connID.String(), "taskID", taskID.String())
			m.EXPECT().SchedulesGet(gomock.Any(), taskID.String(), connID).Return(
				&models.Schedule{}, fmt.Errorf("task get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest("connectorID", connID.String(), "taskID", taskID.String())
			m.EXPECT().SchedulesGet(gomock.Any(), taskID.String(), connID).Return(
				&models.Schedule{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
