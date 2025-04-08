package activities_test

import (
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
	legacy_gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/api/workflowservicemock/v1"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ = Describe("Temporal Workflow Executions List", func() {
	var (
		act        activities.Activities
		ctrl       *gomock.Controller
		legacyCtrl *legacy_gomock.Controller
		t          *activities.MockClient
		w          *workflowservicemock.MockWorkflowServiceClient
		p          *plugins.MockPlugins
		s          *storage.MockStorage
		evts       *events.Events
		logger     logging.Logger
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		legacyCtrl = legacy_gomock.NewController(GinkgoT())

		p = plugins.NewMockPlugins(ctrl)
		t = activities.NewMockClient(ctrl)
		w = workflowservicemock.NewMockWorkflowServiceClient(legacyCtrl)
		s = storage.NewMockStorage(ctrl)
		logger = logging.NewDefaultLogger(GinkgoWriter, true, false, false)
		act = activities.New(logger, t, s, evts, p, time.Millisecond)
	})

	It("returns an error when list workflow execution call fails", func(ctx SpecContext) {
		t.EXPECT().WorkflowService().Return(w)

		req := &workflowservice.ListWorkflowExecutionsRequest{}

		expectedErr := fmt.Errorf("some error")
		w.EXPECT().ListWorkflowExecutions(ctx, gomock.Any()).Return(nil, expectedErr)
		_, err := act.TemporalWorkflowExecutionsList(ctx, req)
		Expect(err).NotTo(BeNil())
	})

	It("returns a list of workflow executions", func(ctx SpecContext) {
		t.EXPECT().WorkflowService().Return(w)

		req := &workflowservice.ListWorkflowExecutionsRequest{}
		expectedRes := &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{StartTime: timestamppb.Now()},
			},
		}

		w.EXPECT().ListWorkflowExecutions(ctx, gomock.Any()).Return(expectedRes, nil)
		res, err := act.TemporalWorkflowExecutionsList(ctx, req)
		Expect(err).To(BeNil())
		Expect(res.Executions).To(HaveLen(len(expectedRes.Executions)))
	})
})
