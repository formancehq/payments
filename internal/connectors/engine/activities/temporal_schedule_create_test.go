package activities_test

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

type scheduleOptionsMatcher struct {
	scheduleID         string
	triggerImmediately bool
	overlap            enums.ScheduleOverlapPolicy
	jitter             time.Duration
	expectEmptySpec    bool
}

func (s *scheduleOptionsMatcher) Matches(x any) bool {
	opts, ok := x.(client.ScheduleOptions)
	if !ok {
		return false
	}

	if opts.ID != s.scheduleID {
		return false
	}
	if opts.TriggerImmediately != s.triggerImmediately {
		return false
	}
	if opts.Overlap != s.overlap {
		return false
	}
	if opts.Spec.Jitter != s.jitter {
		return false
	}
	if s.expectEmptySpec && len(opts.Spec.Intervals) != 0 {
		return false
	}
	return true
}

func (s *scheduleOptionsMatcher) String() string {
	return fmt.Sprintf("has the expected options %q - trigger immediately: %t", s.scheduleID, s.triggerImmediately)
}

var _ = Describe("Temporal Schedule Creation", func() {
	var (
		act    activities.Activities
		ctrl   *gomock.Controller
		t      *activities.MockClient
		sc     *activities.MockScheduleClient
		p      *connectors.MockManager
		s      *storage.MockStorage
		evts   *events.Events
		logger logging.Logger

		scheduleID string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		p = connectors.NewMockManager(ctrl)
		t = activities.NewMockClient(ctrl)
		sc = activities.NewMockScheduleClient(ctrl)
		s = storage.NewMockStorage(ctrl)
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		act = activities.New(logger, t, s, evts, p, time.Millisecond, 0)
		scheduleID = "scheduleID"
	})

	It("returns an error when create fails", func(ctx SpecContext) {
		t.EXPECT().ScheduleClient().Return(sc)

		createOpts := activities.ScheduleCreateOptions{
			ScheduleID: scheduleID,
			Interval:   &client.ScheduleIntervalSpec{Every: time.Second},
		}

		expectedErr := fmt.Errorf("some error")
		sc.EXPECT().Create(ctx, gomock.Any()).Return(nil, expectedErr)
		err := act.TemporalScheduleCreate(ctx, createOpts)
		Expect(err).NotTo(BeNil())
	})

	It("returns no error when schedule is already running", func(ctx SpecContext) {
		t.EXPECT().ScheduleClient().Return(sc)

		createOpts := activities.ScheduleCreateOptions{
			ScheduleID: scheduleID,
			Interval:   &client.ScheduleIntervalSpec{Every: time.Second},
		}

		expectedErr := fmt.Errorf("%w, some error", temporal.ErrScheduleAlreadyRunning)
		sc.EXPECT().Create(ctx, gomock.Any()).Return(nil, expectedErr)
		err := act.TemporalScheduleCreate(ctx, createOpts)
		Expect(err).To(BeNil())
	})

	It("forwards expected create options to temporal", func(ctx SpecContext) {
		t.EXPECT().ScheduleClient().Return(sc)

		createOpts := activities.ScheduleCreateOptions{
			ScheduleID:         scheduleID,
			Interval:           &client.ScheduleIntervalSpec{Every: 5 * time.Second},
			Overlap:            enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Jitter:             2 * time.Second,
			TriggerImmediately: true,
		}
		sc.EXPECT().Create(ctx, &scheduleOptionsMatcher{
			scheduleID:         createOpts.ScheduleID,
			triggerImmediately: true,
			overlap:            createOpts.Overlap,
			jitter:             createOpts.Jitter,
		}).DoAndReturn(func(_ context.Context, opts client.ScheduleOptions) (client.ScheduleHandle, error) {
			// Non-nil Interval must be carried through to Spec.Intervals so the
			// schedule actually fires on its cadence (guards against regressing
			// the periodic path when the one-shot path was added).
			Expect(opts.Spec.Intervals).To(HaveLen(1))
			Expect(opts.Spec.Intervals[0].Every).To(Equal(5 * time.Second))
			return activities.NewMockScheduleHandle(ctrl), nil
		})
		err := act.TemporalScheduleCreate(ctx, createOpts)
		Expect(err).To(BeNil())
	})

	It("rejects options with no Interval and TriggerImmediately false", func(ctx SpecContext) {
		// Guard against a no-op schedule (never fires). Must NOT call
		// ScheduleClient().Create — the validation returns before the client
		// is used.
		createOpts := activities.ScheduleCreateOptions{
			ScheduleID:         scheduleID,
			Interval:           nil,
			TriggerImmediately: false,
		}
		err := act.TemporalScheduleCreate(ctx, createOpts)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(ContainSubstring("schedule would never fire"))
	})

	It("creates a one-shot schedule with empty Spec.Intervals when Interval is nil", func(ctx SpecContext) {
		t.EXPECT().ScheduleClient().Return(sc)

		createOpts := activities.ScheduleCreateOptions{
			ScheduleID:         scheduleID,
			Interval:           nil,
			TriggerImmediately: true,
			SearchAttributes: map[string]interface{}{
				"PaymentScheduleID": scheduleID,
				"Stack":             "test-stack",
			},
		}
		sc.EXPECT().Create(ctx, &scheduleOptionsMatcher{
			scheduleID:         createOpts.ScheduleID,
			triggerImmediately: true,
			expectEmptySpec:    true,
		}).DoAndReturn(func(_ context.Context, opts client.ScheduleOptions) (client.ScheduleHandle, error) {
			// Typed search attributes must still be populated from the legacy map.
			Expect(opts.Action.(*client.ScheduleWorkflowAction).TypedSearchAttributes.Size()).To(Equal(2))
			return activities.NewMockScheduleHandle(ctrl), nil
		})
		err := act.TemporalScheduleCreate(ctx, createOpts)
		Expect(err).To(BeNil())
	})
})
