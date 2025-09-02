package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/serviceerror"
	gomock "go.uber.org/mock/gomock"
)

var _ = Describe("Temporal Workflow Terminations", func() {
	var (
		act    activities.Activities
		ctrl   *gomock.Controller
		t      *activities.MockClient
		p      *connectors.MockManager
		s      *storage.MockStorage
		evts   *events.Events
		logger logging.Logger

		workflowID string
		runID      string
		reason     string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		p = connectors.NewMockManager(ctrl)
		t = activities.NewMockClient(ctrl)
		s = storage.NewMockStorage(ctrl)
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		act = activities.New(logger, t, s, evts, p, time.Millisecond)

		workflowID = "workflowID"
		runID = "runID"
		reason = "reason"
	})

	It("returns an error when terminate workflow call fails with unhandled error", func(ctx SpecContext) {
		expectedErr := fmt.Errorf("some error")
		t.EXPECT().TerminateWorkflow(ctx, workflowID, runID, reason).Return(expectedErr)
		err := act.TemporalWorkflowTerminate(ctx, workflowID, runID, reason)
		Expect(err).NotTo(BeNil())
	})

	It("ignores workflow not found errors", func(ctx SpecContext) {
		expectedErr := serviceerror.NewNotFound("some message")
		t.EXPECT().TerminateWorkflow(ctx, workflowID, runID, reason).Return(expectedErr)
		err := act.TemporalWorkflowTerminate(ctx, workflowID, runID, reason)
		Expect(err).To(BeNil())
	})

	It("terminates a workflow", func(ctx SpecContext) {
		t.EXPECT().TerminateWorkflow(ctx, workflowID, runID, reason).Return(nil)
		err := act.TemporalWorkflowTerminate(ctx, workflowID, runID, reason)
		Expect(err).To(BeNil())
	})
})
