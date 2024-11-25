package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 WorkflowInstances List", func() {
	var (
		handlerFn http.HandlerFunc
		connID    models.ConnectorID
	)
	BeforeEach(func() {
		connID = models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
	})

	Context("list workflowsInstances", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = workflowsInstancesList(m)
		})

		It("should return an invalid ID error when connectorID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", "invalidvalue")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connID.String())
			m.EXPECT().WorkflowsInstancesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Instance]{}, fmt.Errorf("workflowsInstances list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "connectorID", connID.String())
			m.EXPECT().WorkflowsInstancesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Instance]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
