package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
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

var _ = Describe("TemporalSchedulesPause", func() {
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
		act = activities.New(logger, tc, s, evts, p, time.Millisecond)
	})

	It("pauses a schedule and records it in storage", func(ctx SpecContext) {
		reason := "fetch failed: connection timeout"
		instance := models.Instance{
			ID:         "workflow-id-1",
			ScheduleID: "test-connector-FETCH_ACCOUNTS",
			Error:      pointer.For(reason),
		}

		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, instance.ScheduleID).Return(sh)
		sh.EXPECT().Pause(ctx, client.SchedulePauseOptions{Note: reason}).Return(nil)
		s.EXPECT().SchedulesPause(ctx, instance.ScheduleID, gomock.Any(), reason).Return(nil)

		err := act.TemporalSchedulesPause(ctx, []models.Instance{instance})
		Expect(err).To(BeNil())
	})

	It("uses an empty reason when the instance error is nil", func(ctx SpecContext) {
		instance := models.Instance{
			ID:         "workflow-id-2",
			ScheduleID: "test-connector-FETCH_PAYMENTS",
			Error:      nil,
		}

		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, instance.ScheduleID).Return(sh)
		sh.EXPECT().Pause(ctx, client.SchedulePauseOptions{Note: ""}).Return(nil)
		s.EXPECT().SchedulesPause(ctx, instance.ScheduleID, gomock.Any(), "").Return(nil)

		err := act.TemporalSchedulesPause(ctx, []models.Instance{instance})
		Expect(err).To(BeNil())
	})

	It("pauses all instances in the slice", func(ctx SpecContext) {
		ctrl := gomock.NewController(GinkgoT())
		sh2 := activities.NewMockScheduleHandle(ctrl)

		instances := []models.Instance{
			{ID: "wf-1", ScheduleID: "test-connector-FETCH_ACCOUNTS", Error: pointer.For("error 1")},
			{ID: "wf-2", ScheduleID: "test-connector-FETCH_PAYMENTS", Error: pointer.For("error 2")},
		}

		tc.EXPECT().ScheduleClient().Return(sc).Times(2)
		sc.EXPECT().GetHandle(ctx, instances[0].ScheduleID).Return(sh)
		sc.EXPECT().GetHandle(ctx, instances[1].ScheduleID).Return(sh2)
		sh.EXPECT().Pause(ctx, client.SchedulePauseOptions{Note: "error 1"}).Return(nil)
		sh2.EXPECT().Pause(ctx, client.SchedulePauseOptions{Note: "error 2"}).Return(nil)
		s.EXPECT().SchedulesPause(ctx, instances[0].ScheduleID, gomock.Any(), "error 1").Return(nil)
		s.EXPECT().SchedulesPause(ctx, instances[1].ScheduleID, gomock.Any(), "error 2").Return(nil)

		err := act.TemporalSchedulesPause(ctx, instances)
		Expect(err).To(BeNil())
	})

	It("returns empty result for an empty instance list", func(ctx SpecContext) {
		err := act.TemporalSchedulesPause(ctx, []models.Instance{})
		Expect(err).To(BeNil())
	})

	It("returns an error when Pause fails", func(ctx SpecContext) {
		reason := "some error"
		instance := models.Instance{
			ID:         "workflow-id-3",
			ScheduleID: "test-connector-FETCH_ACCOUNTS",
			Error:      pointer.For(reason),
		}

		expectedErr := fmt.Errorf("temporal pause failed")
		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, instance.ScheduleID).Return(sh)
		sh.EXPECT().Pause(ctx, client.SchedulePauseOptions{Note: reason}).Return(expectedErr)

		err := act.TemporalSchedulesPause(ctx, []models.Instance{instance})
		Expect(err).To(MatchError(expectedErr))
	})

	It("returns an error when storage SchedulesPause fails", func(ctx SpecContext) {
		reason := "some error"
		instance := models.Instance{
			ID:         "workflow-id-4",
			ScheduleID: "test-connector-FETCH_ACCOUNTS",
			Error:      pointer.For(reason),
		}

		expectedErr := fmt.Errorf("storage error")
		tc.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, instance.ScheduleID).Return(sh)
		sh.EXPECT().Pause(ctx, client.SchedulePauseOptions{Note: reason}).Return(nil)
		s.EXPECT().SchedulesPause(ctx, instance.ScheduleID, gomock.Any(), reason).Return(expectedErr)

		err := act.TemporalSchedulesPause(ctx, []models.Instance{instance})
		Expect(err).To(MatchError(expectedErr))
	})
})
