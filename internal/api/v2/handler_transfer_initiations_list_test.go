package v2

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

var _ = Describe("API v2 PaymentInitiations List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list paymentInitiations", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = transferInitiationsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().PaymentInitiationsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.PaymentInitiation]{}, fmt.Errorf("paymentInitiations list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().PaymentInitiationsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.PaymentInitiation]{}, nil,
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
