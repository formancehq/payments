package activities_test

import (
	"context"
	"errors"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/testsuite"
	gomock "go.uber.org/mock/gomock"
)

// batchDeleteTestCase defines a test configuration for batch delete activities
type batchDeleteTestCase struct {
	name           string
	entityName     string
	registerFunc   func(env *testsuite.TestActivityEnvironment, act activities.Activities)
	executeFunc    func(env *testsuite.TestActivityEnvironment, act activities.Activities, connectorID models.ConnectorID) error
	expectBatchFn  func(s *storage.MockStorage, connectorID models.ConnectorID, batchSize int) *gomock.Call
}

var batchDeleteTestCases = []batchDeleteTestCase{
	{
		name:       "StorageAccountsDeleteFromConnectorID",
		entityName: "accounts",
		registerFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities) {
			env.RegisterActivity(act.StorageAccountsDeleteFromConnectorID)
		},
		executeFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities, connectorID models.ConnectorID) error {
			_, err := env.ExecuteActivity(act.StorageAccountsDeleteFromConnectorID, connectorID)
			return err
		},
		expectBatchFn: func(s *storage.MockStorage, connectorID models.ConnectorID, batchSize int) *gomock.Call {
			return s.EXPECT().AccountsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, batchSize)
		},
	},
	{
		name:       "StoragePaymentsDeleteFromConnectorID",
		entityName: "payments",
		registerFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities) {
			env.RegisterActivity(act.StoragePaymentsDeleteFromConnectorID)
		},
		executeFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities, connectorID models.ConnectorID) error {
			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			return err
		},
		expectBatchFn: func(s *storage.MockStorage, connectorID models.ConnectorID, batchSize int) *gomock.Call {
			return s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, batchSize)
		},
	},
	{
		name:       "StorageSchedulesDeleteFromConnectorID",
		entityName: "schedules",
		registerFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities) {
			env.RegisterActivity(act.StorageSchedulesDeleteFromConnectorID)
		},
		executeFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities, connectorID models.ConnectorID) error {
			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, connectorID)
			return err
		},
		expectBatchFn: func(s *storage.MockStorage, connectorID models.ConnectorID, batchSize int) *gomock.Call {
			return s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, batchSize)
		},
	},
	{
		name:       "StorageInstancesDelete",
		entityName: "instances",
		registerFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities) {
			env.RegisterActivity(act.StorageInstancesDelete)
		},
		executeFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities, connectorID models.ConnectorID) error {
			_, err := env.ExecuteActivity(act.StorageInstancesDelete, connectorID)
			return err
		},
		expectBatchFn: func(s *storage.MockStorage, connectorID models.ConnectorID, batchSize int) *gomock.Call {
			return s.EXPECT().InstancesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, batchSize)
		},
	},
	{
		name:       "StorageEventsSentDelete",
		entityName: "events_sent",
		registerFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities) {
			env.RegisterActivity(act.StorageEventsSentDelete)
		},
		executeFunc: func(env *testsuite.TestActivityEnvironment, act activities.Activities, connectorID models.ConnectorID) error {
			_, err := env.ExecuteActivity(act.StorageEventsSentDelete, connectorID)
			return err
		},
		expectBatchFn: func(s *storage.MockStorage, connectorID models.ConnectorID, batchSize int) *gomock.Call {
			return s.EXPECT().EventsSentDeleteFromConnectorIDBatch(gomock.Any(), connectorID, batchSize)
		},
	},
}

var _ = Describe("Batch Delete Activities", func() {
	for _, tc := range batchDeleteTestCases {
		tc := tc // capture range variable

		Describe(tc.name, func() {
			var (
				act         activities.Activities
				ctrl        *gomock.Controller
				p           *connectors.MockManager
				s           *storage.MockStorage
				evts        *events.Events
				publisher   *TestPublisher
				logger      = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
				connectorID models.ConnectorID
				env         *testsuite.TestActivityEnvironment
			)

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				p = connectors.NewMockManager(ctrl)
				s = storage.NewMockStorage(ctrl)
				publisher = newTestPublisher()
				evts = events.New(publisher, "")
				act = activities.New(logger, nil, s, evts, p, 0)

				connectorID = models.ConnectorID{Provider: "test", Reference: uuid.New()}

				ts := &testsuite.WorkflowTestSuite{}
				env = ts.NewTestActivityEnvironment()
				tc.registerFunc(env, act)
			})

			AfterEach(func() {
				publisher.Close()
				ctrl.Finish()
			})

			// Helper to set up batch expectations in order
			setupBatchExpectations := func(returns []int, finalErr error) {
				var prevCall *gomock.Call
				for _, ret := range returns {
					call := tc.expectBatchFn(s, connectorID, 1000).Return(ret, nil)
					if prevCall != nil {
						call.After(prevCall)
					}
					prevCall = call
				}
				finalCall := tc.expectBatchFn(s, connectorID, 1000).Return(0, finalErr)
				if prevCall != nil {
					finalCall.After(prevCall)
				}
			}

			Context("when deleting "+tc.entityName+" in batches", func() {
				It("deletes all "+tc.entityName+" successfully in multiple batches", func() {
					setupBatchExpectations([]int{1000, 1000, 500}, nil)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).NotTo(HaveOccurred())
				})

				It("deletes all "+tc.entityName+" in a single batch", func() {
					setupBatchExpectations([]int{500}, nil)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).NotTo(HaveOccurred())
				})

				It("handles empty result set (no "+tc.entityName+" to delete)", func() {
					tc.expectBatchFn(s, connectorID, 1000).Return(0, nil)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("when storage returns errors", func() {
				It("returns error on first batch failure", func() {
					storageErr := errors.New("database connection lost")
					tc.expectBatchFn(s, connectorID, 1000).Return(0, storageErr)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("database connection lost"))
				})

				It("returns error on subsequent batch failure", func() {
					storageErr := errors.New("deadlock detected")
					gomock.InOrder(
						tc.expectBatchFn(s, connectorID, 1000).Return(1000, nil),
						tc.expectBatchFn(s, connectorID, 1000).Return(0, storageErr),
					)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("deadlock detected"))
				})

				It("returns error after successful batches", func() {
					storageErr := errors.New("transaction rollback")
					gomock.InOrder(
						tc.expectBatchFn(s, connectorID, 1000).Return(1000, nil),
						tc.expectBatchFn(s, connectorID, 1000).Return(800, nil),
						tc.expectBatchFn(s, connectorID, 1000).Return(0, storageErr),
					)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("transaction rollback"))
				})
			})

			Context("when batching behavior", func() {
				It("continues until batch returns 0 rows", func() {
					setupBatchExpectations([]int{1000, 1000, 1000, 1000, 1000}, nil)

					err := tc.executeFunc(env, act, connectorID)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			Context("with different connector IDs", func() {
				It("passes the correct connector ID to storage", func() {
					specificConnectorID := models.ConnectorID{Provider: "stripe", Reference: uuid.New()}

					tc.expectBatchFn(s, specificConnectorID, 1000).Return(0, nil)

					err := tc.executeFunc(env, act, specificConnectorID)
					Expect(err).NotTo(HaveOccurred())
				})
			})
		})
	}
})

// Ensure the test context is valid (required for gomock)
var _ context.Context