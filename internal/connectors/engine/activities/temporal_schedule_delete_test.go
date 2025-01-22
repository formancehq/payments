package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/serviceerror"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Temporal Schedule Deletion", func() {
	var (
		act    activities.Activities
		t      *activities.MockClient
		sc     *activities.MockScheduleClient
		sh     *activities.MockScheduleHandle
		p      *plugins.MockPlugins
		s      *storage.MockStorage
		evts   *events.Events
		logger logging.Logger

		scheduleID string
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		p = plugins.NewMockPlugins(ctrl)
		t = activities.NewMockClient(ctrl)
		sc = activities.NewMockScheduleClient(ctrl)
		sh = activities.NewMockScheduleHandle(ctrl)
		s = storage.NewMockStorage(ctrl)
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		act = activities.New(logger, t, s, evts, p, time.Millisecond)
		scheduleID = "scheduleID"
	})

	It("returns unhandled errors", func(ctx SpecContext) {
		t.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, scheduleID).Return(sh)

		expectedErr := fmt.Errorf("some error")
		sh.EXPECT().Delete(ctx).Return(expectedErr)
		err := act.TemporalScheduleDelete(ctx, scheduleID)
		Expect(err).NotTo(BeNil())
	})

	It("skips schedules that are already completed or otherwise not found", func(ctx SpecContext) {
		scheduleID := "scheduleID"
		t.EXPECT().ScheduleClient().Return(sc)
		sc.EXPECT().GetHandle(ctx, scheduleID).Return(sh)

		expectedErr := serviceerror.NewNotFound("some message")
		sh.EXPECT().Delete(ctx).Return(expectedErr)
		err := act.TemporalScheduleDelete(ctx, scheduleID)
		Expect(err).To(BeNil())
	})
})
