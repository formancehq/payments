package v2

import (
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
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
				&bunpaginate.Cursor[models.PaymentInitiation]{
					Data: []models.PaymentInitiation{
						{
							ID:                   models.PaymentInitiationID{},
							ConnectorID:          models.ConnectorID{},
							Reference:            "test",
							CreatedAt:            time.Now().UTC(),
							ScheduledAt:          time.Now().UTC(),
							Description:          "test",
							Type:                 models.PAYMENT_INITIATION_TYPE_PAYOUT,
							SourceAccountID:      &models.AccountID{},
							DestinationAccountID: &models.AccountID{},
							Amount:               big.NewInt(100),
							Asset:                "EUR/2",
							Metadata: map[string]string{
								"test": "test",
							},
						},
					},
				}, nil,
			)

			m.EXPECT().PaymentInitiationAdjustmentsGetLast(gomock.Any(), models.PaymentInitiationID{}).
				Return(&models.PaymentInitiationAdjustment{
					ID:        models.PaymentInitiationAdjustmentID{},
					CreatedAt: time.Now().UTC(),
					Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED,
					Amount:    big.NewInt(100),
					Asset:     pointer.For("EUR/2"),
					Error:     nil,
					Metadata: map[string]string{
						"test": "test",
					},
				}, nil)

			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusOK, "cursor")
		})
	})
})
