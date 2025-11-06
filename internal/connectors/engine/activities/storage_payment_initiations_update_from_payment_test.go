package activities_test

import (
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Storage Payment Initiations Update From Payment", func() {
	var (
		act          activities.Activities
		p            *connectors.MockManager
		s            *storage.MockStorage
		evts         *events.Events
		listResponse []models.PaymentInitiationID
		logger       = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay        = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		listResponse = []models.PaymentInitiationID{
			{
				Reference: "test",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test",
				},
			},
		}
	})

	Context("storage payment initiations update from payment", func() {
		var (
			paymentID models.PaymentID
			status    models.PaymentStatus
			createdAt time.Time
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay)
			paymentID = models.PaymentID{
				PaymentReference: models.PaymentReference{
					Reference: "test",
					Type:      models.PAYMENT_TYPE_PAYOUT,
				},
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test",
				},
			}
			status = models.PAYMENT_STATUS_SUCCEEDED
			createdAt = time.Now()
		})

		It("sucess", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, models.PaymentInitiationAdjustment{
				ID: models.PaymentInitiationAdjustmentID{
					PaymentInitiationID: listResponse[0],
					CreatedAt:           createdAt,
					Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
				},
				CreatedAt: createdAt,
				Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			}).Return(nil)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(BeNil())
		})

		It("list error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, storage.ErrNotFound)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorage, storage.ErrNotFound)))
		})

		It("upsert error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, models.PaymentInitiationAdjustment{
				ID: models.PaymentInitiationAdjustmentID{
					PaymentInitiationID: listResponse[0],
					CreatedAt:           createdAt,
					Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
				},
				CreatedAt: createdAt,
				Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
			}).Return(storage.ErrNotFound)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorage, storage.ErrNotFound)))
		})
	})
})
