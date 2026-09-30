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
		connectorID  models.ConnectorID
		listResponse []models.PaymentInitiation
		logger       = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay        = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
		connectorID = models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "test",
		}
		listResponse = []models.PaymentInitiation{
			{
				ID: models.PaymentInitiationID{
					Reference:   "test",
					ConnectorID: connectorID,
				},
				Amount:   big.NewInt(1000),
				Asset:    "EUR/2",
				Metadata: map[string]string{"key": "value"},
			},
		}
	})

	Context("storage payment initiations update from payment", func() {
		var (
			paymentID models.PaymentID
			status    models.PaymentStatus
			createdAt time.Time
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
				ConnectorID: connectorID,
			}
			status = models.PAYMENT_STATUS_SUCCEEDED
			createdAt = time.Now()
			asset = listResponse[0].Asset
			metadata = listResponse[0].Metadata
		})

		It("success", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, models.PaymentInitiationAdjustment{
				ID: models.PaymentInitiationAdjustmentID{
					PaymentInitiationID: listResponse[0].ID,
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

		It("upserts one adjustment per related payment initiation", func(ctx SpecContext) {
			second := models.PaymentInitiation{
				ID: models.PaymentInitiationID{
					Reference:   "test2",
					ConnectorID: connectorID,
				},
				Amount:   big.NewInt(2000),
				Asset:    "USD/2",
				Metadata: map[string]string{"other": "value"},
			}
			s.EXPECT().PaymentInitiationsListFromPaymentID(ctx, paymentID).Return(append(listResponse, second), nil)

			upserted := make([]models.PaymentInitiationAdjustment, 0, 2)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, gomock.Any()).Times(2).DoAndReturn(
				func(_ any, adj models.PaymentInitiationAdjustment) error {
					upserted = append(upserted, adj)
					return nil
				})

			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(BeNil())
			Expect(upserted).To(HaveLen(2))
			Expect(upserted[0].ID.PaymentInitiationID).To(Equal(listResponse[0].ID))
			Expect(upserted[0].Amount).To(Equal(big.NewInt(1000)))
			Expect(*upserted[0].Asset).To(Equal("EUR/2"))
			Expect(upserted[1].ID.PaymentInitiationID).To(Equal(second.ID))
			Expect(upserted[1].Amount).To(Equal(big.NewInt(2000)))
			Expect(*upserted[1].Asset).To(Equal("USD/2"))
		})

		It("does not upsert when no adjustment is needed", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT, createdAt, paymentID)
			Expect(err).To(BeNil())
		})

		It("list error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationsListFromPaymentID(ctx, paymentID).Return(nil, storage.ErrNotFound)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorageNotFound, storage.ErrNotFound)))
		})

		It("upsert error", func(ctx SpecContext) {
			s.EXPECT().PaymentInitiationsListFromPaymentID(ctx, paymentID).Return(listResponse, nil)
			s.EXPECT().PaymentInitiationAdjustmentsUpsert(ctx, gomock.Any()).Return(storage.ErrNotFound)
			err := act.StoragePaymentInitiationUpdateFromPayment(ctx, status, createdAt, paymentID)
			Expect(err).To(MatchError(temporal.NewNonRetryableApplicationError(storage.ErrNotFound.Error(), activities.ErrTypeStorageNotFound, storage.ErrNotFound)))
		})
	})
})
