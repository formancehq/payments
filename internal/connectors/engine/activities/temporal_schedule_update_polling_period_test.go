package activities_test

import (
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Temporal Schedule Update Polling Period", func() {
	var (
		act    activities.Activities
		p      *connectors.MockManager
		s      *storage.MockStorage
		t      *activities.MockClient
		sc     *activities.MockScheduleClient
		sh     *activities.MockScheduleHandle
		evts   *events.Events
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		delay  = 50 * time.Millisecond
	)

	BeforeEach(func() {
		evts = &events.Events{}
	})

	Context("updating the polling period of a schedule", func() {
		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			p = connectors.NewMockManager(ctrl)
			s = storage.NewMockStorage(ctrl)
			t = activities.NewMockClient(ctrl)
			sc = activities.NewMockScheduleClient(ctrl)
			sh = activities.NewMockScheduleHandle(ctrl)
			act = activities.New(logger, t, s, evts, p, delay)
		})

		It("calls underlying schedule update function", func(ctx SpecContext) {
			t.EXPECT().ScheduleClient().Return(sc)
			sc.EXPECT().GetHandle(gomock.Any(), gomock.Any()).Return(sh)
			sh.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			err := act.TemporalScheduleUpdatePollingPeriod(ctx, "scheduleID", time.Hour)
			Expect(err).To(BeNil())
		})
	})
})
