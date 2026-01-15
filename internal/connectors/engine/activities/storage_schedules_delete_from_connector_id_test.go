package activities_test

import (
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

var _ = Describe("Activity StorageSchedulesDeleteFromConnectorID", func() {
	var (
		act         activities.Activities
		p           *connectors.MockManager
		s           *storage.MockStorage
		evts        *events.Events
		publisher   *TestPublisher
		logger      = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		connectorID models.ConnectorID
		env         *testsuite.TestActivityEnvironment
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		p = connectors.NewMockManager(ctrl)
		s = storage.NewMockStorage(ctrl)
		publisher = newTestPublisher()
		evts = events.New(publisher, "")
		act = activities.New(logger, nil, s, evts, p, 0)

		connectorID = models.ConnectorID{Provider: "test", Reference: uuid.New()}

		ts := &testsuite.WorkflowTestSuite{}
		env = ts.NewTestActivityEnvironment()
		env.RegisterActivity(act.StorageSchedulesDeleteFromConnectorID)
	})

	AfterEach(func() {
		publisher.Close()
	})

	Context("when deleting schedules in batches", func() {
		It("deletes all schedules successfully in multiple batches", func() {
			// Mock three batch deletions: 1000, 1000, 500 (total 2500 schedules)
			gomock.InOrder(
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(500, nil),
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes all schedules in a single batch when count is less than batch size", func() {
			gomock.InOrder(
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(500, nil),
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles empty result set (no schedules to delete)", func() {
			s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil)

			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when storage returns errors", func() {
		It("returns error on first batch failure", func() {
			storageErr := errors.New("database connection lost")
			s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, storageErr)

			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, connectorID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database connection lost"))
		})

		It("returns error on subsequent batch failure", func() {
			storageErr := errors.New("deadlock detected")

			gomock.InOrder(
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, storageErr),
			)

			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, connectorID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deadlock detected"))
		})
	})

	Context("with different connector IDs", func() {
		It("passes the correct connector ID to storage", func() {
			specificConnectorID := models.ConnectorID{Provider: "stripe", Reference: uuid.New()}

			s.EXPECT().SchedulesDeleteFromConnectorIDBatch(gomock.Any(), specificConnectorID, 1000).Return(0, nil)

			_, err := env.ExecuteActivity(act.StorageSchedulesDeleteFromConnectorID, specificConnectorID)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
