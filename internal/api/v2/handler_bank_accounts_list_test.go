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
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v2 Bank Accounts List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list bank accounts", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = bankAccountsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().BankAccountsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.BankAccount]{}, fmt.Errorf("bank accounts list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().BankAccountsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.BankAccount]{
					Data: []models.BankAccount{
						{
							ID:            uuid.New(),
							CreatedAt:     time.Now().UTC(),
							Name:          "test",
							AccountNumber: pointer.For("123456"),
							IBAN:          pointer.For("DE89370400440532013000"),
							SwiftBicCode:  pointer.For("COBADEFF"),
							Country:       pointer.For("DE"),
							Metadata: map[string]string{
								"test": "test",
							},
							RelatedAccounts: []models.BankAccountRelatedAccount{
								{
									AccountID: models.AccountID{
										Reference:   "test",
										ConnectorID: models.ConnectorID{},
									},
									CreatedAt: time.Now().UTC(),
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
