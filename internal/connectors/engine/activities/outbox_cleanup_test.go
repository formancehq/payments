package activities_test

import (
	"context"
	"errors"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	internalevents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("OutboxDeleteOldProcessedEvents", func() {
	var (
		act    activities.Activities
		s      *storage.MockStorage
		evts   *internalevents.Events
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		s = storage.NewMockStorage(ctrl)
		evts = internalevents.New(nil, "http://localhost")
		act = activities.New(logger, nil, s, evts, nil, 0)
	})

	Context("when deleting old processed outbox events", func() {
		It("successfully deletes events older than 1 month", func(ctx SpecContext) {
			// The activity calculates cutoff date as 1 month ago
			// We expect the storage method to be called with a time approximately 1 month ago
			s.EXPECT().
				OutboxEventsDeleteOldProcessed(ctx, gomock.Any()).
				Do(func(_ context.Context, cutoffDate time.Time) {
					// Verify the cutoff date is approximately 1 month ago (within 1 day tolerance)
					expectedCutoff := time.Now().UTC().AddDate(0, -1, 0)
					diff := expectedCutoff.Sub(cutoffDate)
					Expect(diff).To(BeNumerically("<", 24*time.Hour))
					Expect(diff).To(BeNumerically(">", -24*time.Hour))
				}).
				Return(nil)

			err := act.OutboxDeleteOldProcessedEvents(ctx)
			Expect(err).To(BeNil())
		})

		It("handles storage error", func(ctx SpecContext) {
			expectedErr := errors.New("database error")
			s.EXPECT().
				OutboxEventsDeleteOldProcessed(ctx, gomock.Any()).
				Return(expectedErr)

			err := act.OutboxDeleteOldProcessedEvents(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to delete old processed outbox events"))
		})
	})
})
