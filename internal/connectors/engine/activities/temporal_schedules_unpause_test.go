package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/sdk/client"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("TemporalSchedulesUnpause", func() {
	var (
		act    activities.Activities
		tc     *activities.MockClient
		sc     *activities.MockScheduleClient
		sh     *activities.MockScheduleHandle
		p      *connectors.MockManager
		s      *storage.MockStorage
		evts   *events.Events
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		p = connectors.NewMockManager(ctrl)
		tc = activities.NewMockClient(ctrl)
		sc = activities.NewMockScheduleClient(ctrl)
		sh = activities.NewMockScheduleHandle(ctrl)
		s = storage.NewMockStorage(ctrl)
		evts = &events.Events{}
		act = activities.New(logger, tc, s, evts, p, time.Millisecond, 0)
	})

	It("unpauses a schedule and clears it in storage", func(ctx SpecContext) {
		schedule := models.Schedule{
			ID:          "test-connector-FETCH_ACCOUNTS",
			ConnectorID: models.ConnectorID{Reference: [16]byte{1}, Provider: "test"},
		}

		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, schedule.ID).Return(sh)
		sh.EXPECT().Unpause(ctx, client.ScheduleUnpauseOptions{}).Return(nil)
		s.EXPECT().SchedulesUnpause(ctx, schedule.ID, schedule.ConnectorID).Return(nil)

		err := act.TemporalSchedulesUnpause(ctx, []models.Schedule{schedule})
		Expect(err).To(BeNil())
	})

	It("unpauses all schedules in the slice", func(ctx SpecContext) {
		ctrl := gomock.NewController(GinkgoT())
		sh2 := activities.NewMockScheduleHandle(ctrl)

		schedules := []models.Schedule{
			{ID: "test-connector-FETCH_ACCOUNTS"},
			{ID: "test-connector-FETCH_PAYMENTS"},
		}

		tc.EXPECT().ScheduleClient().Return(sc).Times(2)
		sc.EXPECT().GetHandle(ctx, schedules[0].ID).Return(sh)
		sc.EXPECT().GetHandle(ctx, schedules[1].ID).Return(sh2)
		sh.EXPECT().Unpause(ctx, client.ScheduleUnpauseOptions{}).Return(nil)
		sh2.EXPECT().Unpause(ctx, client.ScheduleUnpauseOptions{}).Return(nil)
		s.EXPECT().SchedulesUnpause(ctx, schedules[0].ID, schedules[0].ConnectorID).Return(nil)
		s.EXPECT().SchedulesUnpause(ctx, schedules[1].ID, schedules[1].ConnectorID).Return(nil)

		err := act.TemporalSchedulesUnpause(ctx, schedules)
		Expect(err).To(BeNil())
	})

	It("returns nil for an empty slice", func(ctx SpecContext) {
		err := act.TemporalSchedulesUnpause(ctx, []models.Schedule{})
		Expect(err).To(BeNil())
	})

	It("returns an error when Unpause fails", func(ctx SpecContext) {
		schedule := models.Schedule{ID: "test-connector-FETCH_ACCOUNTS"}

		expectedErr := fmt.Errorf("temporal unpause failed")
		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, schedule.ID).Return(sh)
		sh.EXPECT().Unpause(ctx, client.ScheduleUnpauseOptions{}).Return(expectedErr)

		err := act.TemporalSchedulesUnpause(ctx, []models.Schedule{schedule})
		Expect(err).To(MatchError(expectedErr))
	})

	It("returns an error when storage SchedulesUnpause fails", func(ctx SpecContext) {
		schedule := models.Schedule{ID: "test-connector-FETCH_ACCOUNTS"}

		expectedErr := fmt.Errorf("storage error")
		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, schedule.ID).Return(sh)
		sh.EXPECT().Unpause(ctx, client.ScheduleUnpauseOptions{}).Return(nil)
		s.EXPECT().SchedulesUnpause(ctx, schedule.ID, schedule.ConnectorID).Return(expectedErr)

		err := act.TemporalSchedulesUnpause(ctx, []models.Schedule{schedule})
		Expect(err).To(MatchError(expectedErr))
	})
})
