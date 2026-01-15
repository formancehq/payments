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

var _ = Describe("Activity StoragePaymentsDeleteFromConnectorID", func() {
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
		env.RegisterActivity(act.StoragePaymentsDeleteFromConnectorID)
	})

	AfterEach(func() {
		publisher.Close()
	})

	Context("when deleting payments in batches", func() {
		It("deletes all payments successfully in multiple batches", func() {
			// Mock three batch deletions: 1000, 1000, 500 (total 2500 payments)
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(500, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes all payments in a single batch when count is less than batch size", func() {
			// Only 500 payments, all deleted in one batch
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(500, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles empty result set (no payments to delete)", func() {
			s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("processes exactly batch size of 1000", func() {
			// Verify the activity uses the correct batch size
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when storage returns errors", func() {
		It("returns error on first batch failure", func() {
			storageErr := errors.New("database connection lost")
			s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, storageErr)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("database connection lost"))
		})

		It("returns error on subsequent batch failure", func() {
			storageErr := errors.New("deadlock detected")

			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, storageErr),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deadlock detected"))
		})

		It("returns error after successful batches", func() {
			storageErr := errors.New("transaction rollback")

			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(800, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, storageErr),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("transaction rollback"))
		})
	})

	Context("when batching behavior", func() {
		It("continues until batch returns 0 rows", func() {
			// Mock batches that keep returning data until exhausted
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles partial final batch", func() {
			// Last batch has fewer than 1000 items
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(1000, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(123, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connectorID, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connectorID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with different connector IDs", func() {
		It("passes the correct connector ID to storage", func() {
			specificConnectorID := models.ConnectorID{Provider: "stripe", Reference: uuid.New()}

			s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), specificConnectorID, 1000).Return(0, nil)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, specificConnectorID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles multiple connectors independently", func() {
			connector1 := models.ConnectorID{Provider: "stripe", Reference: uuid.New()}
			connector2 := models.ConnectorID{Provider: "wise", Reference: uuid.New()}

			// First connector
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connector1, 1000).Return(500, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connector1, 1000).Return(0, nil),
			)

			_, err := env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connector1)
			Expect(err).NotTo(HaveOccurred())

			// Second connector
			gomock.InOrder(
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connector2, 1000).Return(300, nil),
				s.EXPECT().PaymentsDeleteFromConnectorIDBatch(gomock.Any(), connector2, 1000).Return(0, nil),
			)

			_, err = env.ExecuteActivity(act.StoragePaymentsDeleteFromConnectorID, connector2)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
