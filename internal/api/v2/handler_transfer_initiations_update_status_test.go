package v2

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Transfer Initiation Update Status", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("update payment initiation", func() {
		var (
			w    *httptest.ResponseRecorder
			m    *backend.MockBackend
			utsr updateTransferInitiationStatusRequest
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = transferInitiationsUpdateStatus(m)
		})

		It("should return a bad request error when payment init id invalid", func(ctx SpecContext) {
			req := prepareJSONRequestWithQuery(http.MethodPost, "transferInitiationID", "invalid", nil)
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		DescribeTable("validation errors",
			func(r updateTransferInitiationStatusRequest) {
				handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "transferInitiationID", paymentID.String(), &r))
				assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrValidation)
			},
			Entry("status missing", updateTransferInitiationStatusRequest{}),
			Entry("status invalid", updateTransferInitiationStatusRequest{Status: "invalid"}),
		)

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation update err")
			m.EXPECT().PaymentInitiationsApprove(gomock.Any(), gomock.Any(), true).Return(
				models.Task{},
				expectedErr,
			)
			utsr = updateTransferInitiationStatusRequest{
				Status: "VALIDATED",
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "transferInitiationID", paymentID.String(), &utsr))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should call approve backend when status is validated", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsApprove(gomock.Any(), gomock.Any(), true).Return(
				models.Task{},
				nil,
			)
			utsr = updateTransferInitiationStatusRequest{
				Status: "VALIDATED",
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "transferInitiationID", paymentID.String(), &utsr))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})

		It("should call reject backend when status is rejected", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsReject(gomock.Any(), gomock.Any()).Return(nil)
			utsr = updateTransferInitiationStatusRequest{
				Status: "REJECTED",
			}
			handlerFn(w, prepareJSONRequestWithQuery(http.MethodPost, "transferInitiationID", paymentID.String(), &utsr))
			assertExpectedResponse(w.Result(), http.StatusNoContent, "")
		})
	})
})
