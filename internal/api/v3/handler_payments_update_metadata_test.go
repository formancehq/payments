package v3

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/golang/mock/gomock"
)

var _ = Describe("API v3 Payments Update Metadata", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		payRef := models.PaymentReference{Reference: "ref", Type: models.PAYMENT_TYPE_TRANSFER}
		paymentID = models.PaymentID{PaymentReference: payRef, ConnectorID: connID}
	})

	Context("update payment metadata", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = paymentsUpdateMetadata(m)
		})

		It("should return a bad request error when paymentID is invalid", func(ctx SpecContext) {
			payload := map[string]string{}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "paymentID", "invalid", &payload))

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		DescribeTable("validation errors",
			func(payload map[string]string) {
				handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "paymentID", paymentID.String(), &payload))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("metadata missing", nil),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment update metadata err")
			m.EXPECT().PaymentsUpdateMetadata(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr)
			metadata := map[string]string{"meta": "data"}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "paymentID", paymentID.String(), &metadata))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status no content on success", func(ctx SpecContext) {
			metadata := map[string]string{"meta": "data"}
			m.EXPECT().PaymentsUpdateMetadata(gomock.Any(), gomock.Any(), metadata).Return(nil)
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPatch, "paymentID", paymentID.String(), &metadata))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
