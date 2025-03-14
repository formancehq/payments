package v3

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v3 Counter Parties List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list counter parties", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = counterPartiesList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().CounterPartiesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.CounterParty]{}, fmt.Errorf("counter parties list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().CounterPartiesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.CounterParty]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
