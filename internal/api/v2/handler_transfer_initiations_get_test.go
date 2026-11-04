package v2

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/pkg/client/models/components"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("API v2 Transfer Initiation Get", func() {
	var (
		handlerFn http.HandlerFunc
		paymentID models.PaymentInitiationID
	)
	BeforeEach(func() {
		connID := models.ConnectorID{Reference: uuid.New(), Provider: "psp"}
		paymentID = models.PaymentInitiationID{Reference: "ref", ConnectorID: connID}
	})

	Context("get payment initiation", func() {
		var (
			w *httptest.ResponseRecorder
			m *backend.MockBackend
		)
		BeforeEach(func() {
			w = httptest.NewRecorder()
			ctrl := gomock.NewController(GinkgoT())
			m = backend.NewMockBackend(ctrl)
			handlerFn = transferInitiationsGet(m)
		})

		It("should return a bad request error when transferInitiationID is invalid", func(ctx SpecContext) {
			req := prepareQueryRequest(http.MethodGet, "transferInitiationID", "invalid")
			handlerFn(w, req)

			assertExpectedResponse(w.Result(), http.StatusBadRequest, ErrInvalidID)
		})

		It("should return an internal server error when backend returns error", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation get err")
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), gomock.Any()).Return(
				&models.PaymentInitiation{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "transferInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return an internal server error when backend returns error fetching payments", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation get payments err")
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), gomock.Any()).Return(
				&models.PaymentInitiation{},
				nil,
			)
			m.EXPECT().PaymentInitiationRelatedPaymentsListAll(gomock.Any(), gomock.Any()).Return(
				[]models.Payment{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "transferInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return an internal server error when backend returns error fetching payment adjustments", func(ctx SpecContext) {
			expectedErr := errors.New("payment initiation get adjustments err")
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), gomock.Any()).Return(
				&models.PaymentInitiation{},
				nil,
			)
			m.EXPECT().PaymentInitiationRelatedPaymentsListAll(gomock.Any(), gomock.Any()).Return(
				[]models.Payment{},
				nil,
			)
			m.EXPECT().PaymentInitiationAdjustmentsListAll(gomock.Any(), paymentID).Return(
				[]models.PaymentInitiationAdjustment{},
				expectedErr,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "transferInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusInternalServerError, "INTERNAL")
		})

		It("should return status ok on success", func(ctx SpecContext) {
			m.EXPECT().PaymentInitiationsGet(gomock.Any(), paymentID).Return(
				&models.PaymentInitiation{},
				nil,
			)
			m.EXPECT().PaymentInitiationRelatedPaymentsListAll(gomock.Any(), gomock.Any()).Return(
				[]models.Payment{},
				nil,
			)
			m.EXPECT().PaymentInitiationAdjustmentsListAll(gomock.Any(), paymentID).Return(
				[]models.PaymentInitiationAdjustment{},
				nil,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "transferInitiationID", paymentID.String()))
			assertExpectedResponse(w.Result(), http.StatusOK, "data")
		})

		It("should leave dropped SCHEDULED_FOR_PROCESSING adjustments out of relatedAdjustments", func(ctx SpecContext) {
			now := time.Now().UTC().Truncate(time.Second)
			asset := "USD/2"
			adjustment := func(status models.PaymentInitiationAdjustmentStatus, createdAt time.Time) models.PaymentInitiationAdjustment {
				return models.PaymentInitiationAdjustment{
					ID: models.PaymentInitiationAdjustmentID{
						PaymentInitiationID: paymentID,
						CreatedAt:           createdAt,
						Status:              status,
					},
					CreatedAt: createdAt,
					Status:    status,
					Amount:    big.NewInt(100),
					Asset:     &asset,
				}
			}

			m.EXPECT().PaymentInitiationsGet(gomock.Any(), paymentID).Return(
				&models.PaymentInitiation{ID: paymentID, ConnectorID: paymentID.ConnectorID, Type: models.PAYMENT_INITIATION_TYPE_TRANSFER, Amount: big.NewInt(100), Asset: asset},
				nil,
			)
			m.EXPECT().PaymentInitiationRelatedPaymentsListAll(gomock.Any(), gomock.Any()).Return(
				[]models.Payment{},
				nil,
			)
			// A scheduled transfer: the workflow records SCHEDULED_FOR_PROCESSING
			// first, then PROCESSING once the scheduled time is reached (newest first).
			m.EXPECT().PaymentInitiationAdjustmentsListAll(gomock.Any(), paymentID).Return(
				[]models.PaymentInitiationAdjustment{
					adjustment(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING, now),
					adjustment(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING, now.Add(-time.Minute)),
				},
				nil,
			)
			handlerFn(w, prepareQueryRequest(http.MethodGet, "transferInitiationID", paymentID.String()))

			res := w.Result()
			defer res.Body.Close()
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			var body struct {
				Data struct {
					RelatedAdjustments []transferInitiationAdjustmentsResponse `json:"relatedAdjustments"`
				} `json:"data"`
			}
			raw := w.Body.Bytes()
			Expect(json.Unmarshal(raw, &body)).To(Succeed())
			Expect(body.Data.RelatedAdjustments).To(HaveLen(1))
			Expect(body.Data.RelatedAdjustments[0].Status).To(Equal("PROCESSING"))

			// The generated v2 SDK must be able to decode the response.
			var sdkResponse components.TransferInitiationResponse
			Expect(json.Unmarshal(raw, &sdkResponse)).To(Succeed())
		})
	})
})
