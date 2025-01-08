package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Pools List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list pools", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = poolsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().PoolsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Pool]{}, fmt.Errorf("pools list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().PoolsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Pool]{
					Data: []models.Pool{
						{
							ID:        uuid.New(),
							Name:      "test",
							CreatedAt: time.Now().UTC(),
							PoolAccounts: []models.AccountID{
								{
									Reference:   "test",
									ConnectorID: models.ConnectorID{},
								},
							},
						},
					},
				}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
