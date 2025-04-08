package v2

import (
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Payments List", func() {
	var (
		handlerFn http.HandlerFunc
	)

	Context("list payments", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentsList(m)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().PaymentsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Payment]{}, fmt.Errorf("payments list error"),
			)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return a cursor object", func(ctx SpecContext) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			m.EXPECT().PaymentsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Payment]{
					Data: []models.Payment{
						{
							ID:                   models.PaymentID{},
							ConnectorID:          models.ConnectorID{},
							Reference:            "test",
							CreatedAt:            time.Now().UTC(),
							Type:                 models.PAYMENT_TYPE_PAYIN,
							InitialAmount:        big.NewInt(100),
							Amount:               big.NewInt(100),
							Asset:                "EUR/2",
							Scheme:               models.PAYMENT_SCHEME_A2A,
							Status:               models.PAYMENT_STATUS_CAPTURE_FAILED,
							SourceAccountID:      &models.AccountID{},
							DestinationAccountID: &models.AccountID{},
							Metadata: map[string]string{
								"test": "test",
							},
							Adjustments: []models.PaymentAdjustment{
								{
									ID:        models.PaymentAdjustmentID{},
									Reference: "test",
									CreatedAt: time.Now().UTC(),
									Status:    models.PAYMENT_STATUS_AMOUNT_ADJUSTEMENT,
									Amount:    big.NewInt(100),
									Asset:     pointer.For("EUR/2"),
									Metadata: map[string]string{
										"test": "test",
									},
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
