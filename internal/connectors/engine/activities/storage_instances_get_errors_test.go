package activities_test

import (
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
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

var _ = Describe("Activity StorageInstancesGetScheduleErrors", func() {
	var (
		act       activities.Activities
		ctrl      *gomock.Controller
		p         *connectors.MockManager
		s         *storage.MockStorage
		evts      *events.Events
		publisher *TestPublisher
		logger    = logging.NewDefaultLogger(GinkgoWriter, true, false, false)

		connectorID             models.ConnectorID
		now                     time.Time
		healthCheckErrorThreshold = 5
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		p = connectors.NewMockManager(ctrl)
		s = storage.NewMockStorage(ctrl)
		publisher = newTestPublisher()
		evts = events.New(publisher, "")

		act = activities.New(logger, nil, s, evts, p, 0, healthCheckErrorThreshold)

		connectorID = models.ConnectorID{Provider: "test", Reference: uuid.New()}
		now = time.Now().UTC()
	})

	AfterEach(func() {
		publisher.Close()
		ctrl.Finish()
	})

	Context("when cursor is nil", func() {
		It("builds a default query and returns results", func(ctx SpecContext) {
			expected := &bunpaginate.Cursor[models.Instance]{
				Data: []models.Instance{{
					ID:          "inst-1",
					ScheduleID:  "sched-1",
					ConnectorID: connectorID,
					CreatedAt:   now,
					UpdatedAt:   now,
				}},
			}
			s.EXPECT().
				InstancesGetScheduleErrors(gomock.Any(), connectorID, gomock.Any(), healthCheckErrorThreshold).
				Return(expected, nil)

			result, err := act.StorageInstancesGetScheduleErrors(ctx, connectorID, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("when cursor is a pointer to an empty string", func() {
		It("builds a default query and returns results", func(ctx SpecContext) {
			empty := ""
			expected := &bunpaginate.Cursor[models.Instance]{Data: []models.Instance{}}
			s.EXPECT().
				InstancesGetScheduleErrors(gomock.Any(), connectorID, gomock.Any(), healthCheckErrorThreshold).
				Return(expected, nil)

			result, err := act.StorageInstancesGetScheduleErrors(ctx, connectorID, &empty)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("when cursor is a valid encoded string", func() {
		It("unmarshals the cursor and calls storage with the decoded query", func(ctx SpecContext) {
			q := storage.NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(storage.InstanceQuery{}))
			cursorStr := bunpaginate.EncodeCursor(q)

			expected := &bunpaginate.Cursor[models.Instance]{
				Data: []models.Instance{{ID: "inst-2", ConnectorID: connectorID, CreatedAt: now, UpdatedAt: now}},
			}
			s.EXPECT().
				InstancesGetScheduleErrors(gomock.Any(), connectorID, gomock.Any(), healthCheckErrorThreshold).
				Return(expected, nil)

			result, err := act.StorageInstancesGetScheduleErrors(ctx, connectorID, &cursorStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("when cursor is an invalid encoded string", func() {
		It("returns an error without calling storage", func(ctx SpecContext) {
			invalid := "not!!valid!!base64"

			result, err := act.StorageInstancesGetScheduleErrors(ctx, connectorID, &invalid)
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("when storage returns an error", func() {
		It("wraps and returns the error", func(ctx SpecContext) {
			s.EXPECT().
				InstancesGetScheduleErrors(gomock.Any(), connectorID, gomock.Any(), healthCheckErrorThreshold).
				Return(nil, errors.New("db failure"))

			result, err := act.StorageInstancesGetScheduleErrors(ctx, connectorID, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("db failure"))
			Expect(result).To(BeNil())
		})
	})

	Context("when storage returns multiple instances with HasMore", func() {
		It("returns the full cursor unchanged", func(ctx SpecContext) {
			errMsg := "schedule timed out"
			expected := &bunpaginate.Cursor[models.Instance]{
				HasMore: true,
				Data: []models.Instance{
					{
						ID:          "inst-a",
						ScheduleID:  "sched-x",
						ConnectorID: connectorID,
						CreatedAt:   now,
						UpdatedAt:   now,
						Terminated:  true,
						Error:       &errMsg,
					},
					{
						ID:          "inst-b",
						ScheduleID:  "sched-y",
						ConnectorID: connectorID,
						CreatedAt:   now,
						UpdatedAt:   now,
						Terminated:  false,
					},
				},
			}
			s.EXPECT().
				InstancesGetScheduleErrors(gomock.Any(), connectorID, gomock.Any(), healthCheckErrorThreshold).
				Return(expected, nil)

			result, err := act.StorageInstancesGetScheduleErrors(ctx, connectorID, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.HasMore).To(BeTrue())
			Expect(result.Data).To(HaveLen(2))
			Expect(result.Data[0].Error).To(Equal(&errMsg))
			Expect(result.Data[1].Error).To(BeNil())
		})
	})

	Context("when the healthCheckErrorThreshold is passed to storage", func() {
		It("uses the threshold configured at construction time", func(ctx SpecContext) {
			customAct := activities.New(logger, nil, s, evts, p, 0, 10)
			s.EXPECT().
				InstancesGetScheduleErrors(gomock.Any(), connectorID, gomock.Any(), 10).
				Return(&bunpaginate.Cursor[models.Instance]{}, nil)

			_, err := customAct.StorageInstancesGetScheduleErrors(ctx, connectorID, nil)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
