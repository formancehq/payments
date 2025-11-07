package activities_test

import (
	"errors"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Send Events", func() {
	var (
		act       activities.Activities
		p         *connectors.MockManager
		publisher *TestPublisher
		s         *storage.MockStorage
		evts      *events.Events
		logger    = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay     = 50 * time.Millisecond
	)

	Context("plugin complete user link", func() {
		var (
			ik          string
			connectorID models.ConnectorID
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			publisher = newTestPublisher()
			evts = events.New(publisher, "")
			act = activities.New(logger, nil, s, evts, p, delay)

			ik = "test"
			connectorID = models.ConnectorID{
				Provider:  "test",
				Reference: uuid.New(),
			}
		})

		AfterEach(func() {
			publisher.Close()
		})

		It("should not do anything if the event was already sent", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(true, nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).NotTo(Receive())
		})

		It("should fail if storage fails", func(ctx SpecContext) {
			at := time.Now().UTC()

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, errors.New("test"))

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
			})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("test")))
		})

		It("should send an account event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				Account: &models.Account{
					ID:          models.AccountID{},
					ConnectorID: connectorID,
					Connector: &models.ConnectorBase{
						ID:        connectorID,
						Name:      "test",
						CreatedAt: time.Now().UTC(),
						Provider:  "test",
					},
					Reference: "test",
					CreatedAt: time.Now().UTC(),
					Type:      models.ACCOUNT_TYPE_INTERNAL,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a balance event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				Balance: &models.Balance{
					AccountID: models.AccountID{
						Reference:   "test",
						ConnectorID: connectorID,
					},
					Asset:   "USD/2",
					Balance: big.NewInt(100),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a bank account event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				BankAccount: &models.BankAccount{
					ID:        uuid.New(),
					CreatedAt: time.Now().UTC(),
					Name:      "test",
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a payment event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				Payment: &activities.SendEventsPayment{
					Payment: models.Payment{
						ID: models.PaymentID{
							PaymentReference: models.PaymentReference{
								Reference: "test",
								Type:      models.PAYMENT_TYPE_PAYIN,
							},
							ConnectorID: connectorID,
						},
						ConnectorID: connectorID,
						Reference:   "test",
						CreatedAt:   time.Now().UTC(),
						Type:        models.PAYMENT_TYPE_PAYIN,
						Amount:      big.NewInt(100),
						Asset:       "USD/2",
						Status:      models.PAYMENT_STATUS_SUCCEEDED,
					},
					Adjustment: models.PaymentAdjustment{
						ID: models.PaymentAdjustmentID{
							PaymentID: models.PaymentID{
								PaymentReference: models.PaymentReference{
									Reference: "test",
									Type:      models.PAYMENT_TYPE_PAYIN,
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a payment deleted event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				PaymentDeleted: &models.PaymentID{
					PaymentReference: models.PaymentReference{
						Reference: "test",
						Type:      models.PAYMENT_TYPE_PAYIN,
					},
					ConnectorID: connectorID,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a connector reset event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				ConnectorReset: &connectorID,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a pool creation event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				PoolsCreation: &models.Pool{
					ID: uuid.New(),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a pool deletion event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				PoolsDeletion:  pointer.For(uuid.New()),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a payment initiation event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				PaymentInitiation: &models.PaymentInitiation{
					ID: models.PaymentInitiationID{
						Reference:   "test",
						ConnectorID: connectorID,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a payment initiation adjustment event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				PaymentInitiationAdjustment: &models.PaymentInitiationAdjustment{
					ID: models.PaymentInitiationAdjustmentID{
						PaymentInitiationID: models.PaymentInitiationID{
							Reference:   "test",
							ConnectorID: connectorID,
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a payment initiation related payment event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				PaymentInitiationRelatedPayment: &models.PaymentInitiationRelatedPayments{
					PaymentInitiationID: models.PaymentInitiationID{
						Reference:   "test",
						ConnectorID: connectorID,
					},
					PaymentID: models.PaymentID{
						PaymentReference: models.PaymentReference{
							Reference: "test",
							Type:      models.PAYMENT_TYPE_PAYIN,
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a user pending disconnect event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				UserPendingDisconnect: &models.UserConnectionPendingDisconnect{
					PsuID:        uuid.New(),
					ConnectorID:  connectorID,
					ConnectionID: "test",
					At:           time.Now().UTC(),
					Reason:       pointer.For("test"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a user disconnected event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				UserDisconnected: &models.UserDisconnected{
					PsuID:       uuid.New(),
					ConnectorID: connectorID,
					At:          time.Now().UTC(),
					Reason:      pointer.For("test"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a user connection disconnected event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				UserConnectionDisconnected: &models.UserConnectionDisconnected{
					PsuID:        uuid.New(),
					ConnectorID:  connectorID,
					ConnectionID: "test",
					At:           time.Now().UTC(),
					Reason:       pointer.For("test"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a user connection reconnected event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				UserConnectionReconnected: &models.UserConnectionReconnected{
					PsuID:        uuid.New(),
					ConnectorID:  connectorID,
					ConnectionID: "test",
					At:           time.Now().UTC(),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a user link status event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				UserLinkStatus: &models.UserLinkSessionFinished{
					PsuID:       uuid.New(),
					ConnectorID: connectorID,
					AttemptID:   uuid.New(),
					Status:      models.OpenBankingConnectionAttemptStatusCompleted,
					Error:       pointer.For("test"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a user connection data synced event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				UserConnectionDataSynced: &models.UserConnectionDataSynced{
					PsuID:        uuid.New(),
					ConnectorID:  connectorID,
					ConnectionID: "test",
					At:           time.Now().UTC(),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})

		It("should send a task event", func(ctx SpecContext) {
			at := time.Now().UTC()

			respChan := publisher.Subscribe(ctx)

			s.EXPECT().EventsSentExists(gomock.Any(), models.EventID{
				EventIdempotencyKey: ik,
				ConnectorID:         &connectorID,
			}).Return(false, nil)
			s.EXPECT().EventsSentUpsert(gomock.Any(), models.EventSent{
				ID: models.EventID{
					EventIdempotencyKey: ik,
					ConnectorID:         &connectorID,
				},
				ConnectorID: &connectorID,
				SentAt:      at,
			}).Return(nil)

			err := act.SendEvents(ctx, activities.SendEventsRequest{
				ConnectorID:    &connectorID,
				IdempotencyKey: ik,
				At:             at,
				Task: &models.Task{
					ID: models.TaskID{
						Reference:   "test",
						ConnectorID: connectorID,
					},
					ConnectorID:     &connectorID,
					Status:          models.TASK_STATUS_SUCCEEDED,
					CreatedAt:       time.Now().UTC(),
					UpdatedAt:       time.Now().UTC(),
					CreatedObjectID: pointer.For("test"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(respChan).To(Receive())
		})
	})
})
