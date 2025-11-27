package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Accounts List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list accounts", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = accountsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().AccountsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Account]{}, fmt.Errorf("accounts list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			connectorID := models.ConnectorID{
				Reference: uuid.New(),
				Provider:  "test",
			}
			m.EXPECT().AccountsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Account]{
					Data: []models.Account{
						{
							ID: models.AccountID{
								Reference:   "test",
								ConnectorID: connectorID,
							},
							ConnectorID:  connectorID,
							Reference:    "test",
							CreatedAt:    time.Now().UTC(),
							Type:         models.ACCOUNT_TYPE_INTERNAL,
							Name:         pointer.For("test"),
							DefaultAsset: pointer.For("USD/2"),
							Metadata: map[string]string{
								"foo": "bar",
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
