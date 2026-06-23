package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Orders List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list orders", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = ordersList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().OrdersList(gomock.Any(), gomock.Any()).Return(
				&paginate.Cursor[models.Order]{}, fmt.Errorf("orders list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().OrdersList(gomock.Any(), gomock.Any()).Return(
				&paginate.Cursor[models.Order]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
