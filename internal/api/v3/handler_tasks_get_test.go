package v3

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

var _ = Describe("API v3 Get Task", func() {
	var (
		handlerFn http.HandlerFunc
		taskID    models.TaskID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
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

		It("should return an invalid ID error when taskID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "taskID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "taskID", taskID.String())
			m.EXPECT().TaskGet(gomock.Any(), taskID).Return(
				&models.Task{}, fmt.Errorf("task get error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return data object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "taskID", taskID.String())
			m.EXPECT().TaskGet(gomock.Any(), taskID).Return(
				&models.Task{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
