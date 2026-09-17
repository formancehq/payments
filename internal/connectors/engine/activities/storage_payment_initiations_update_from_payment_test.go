package activities_test

import (
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
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
			pi        *models.PaymentInitiation
			asset     string
			metadata  map[string]string
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			act = activities.New(logger, nil, s, evts, p, delay, 0)
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
			asset = "EUR/2"
			metadata = map[string]string{"key": "value"}
			pi = &models.PaymentInitiation{
				ID:       listResponse[0],
				Amount:   big.NewInt(1000),
				Asset:    asset,
				Metadata: metadata,
			}
		})

		It("success", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationsGet(ctx, listResponse[0]).Return(pi, nil)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, models.PaymentInitiationAdjustment{
				ID: models.PaymentInitiationAdjustmentID{
					PaymentInitiationID: listResponse[0],
					CreatedAt:           createdAt,
					Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
				},
				CreatedAt: createdAt,
				Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
				Amount:    big.NewInt(1000),
				Asset:     &asset,
				Metadata:  metadata,
			}).Return(nil)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(BeNil())
		})

		It("skips enrichment when the payment initiation is gone", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationsGet(ctx, listResponse[0]).Return(nil, storage.ErrNotFound)
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

		It("does not fetch the payment initiation when no adjustment is needed", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT, createdAt, paymentID)
			Expect(err).To(BeNil())
		})

		It("list error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, storage.ErrNotFound)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorageNotFound, storage.ErrNotFound)))
		})

		It("get payment initiation error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationsGet(ctx, listResponse[0]).Return(nil, storage.ErrValidation)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).ToNot(BeNil())
		})

		It("upsert error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationIDsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationsGet(ctx, listResponse[0]).Return(pi, nil)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, gomock.Any()).Return(storage.ErrNotFound)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorageNotFound, storage.ErrNotFound)))
		})
	})
})
