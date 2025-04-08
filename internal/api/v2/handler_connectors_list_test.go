package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Connectors List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list connectors", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = connectorsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{}, fmt.Errorf("connectors list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{
					Data: []models.Connector{
						{
							ID:                   models.ConnectorID{},
							Name:                 "test",
							CreatedAt:            time.Now().UTC(),
							Provider:             "test",
							ScheduledForDeletion: false,
							Config:               []byte("{}"),
						},
					},
				}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})
	})
})
