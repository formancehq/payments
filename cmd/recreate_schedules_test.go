package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	gomock "go.uber.org/mock/gomock"
)

func TestRecreateSchedules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RecreateSchedules Suite")
}

var _ = Describe("RecreateSchedules", func() {
	var (
		ctrl               *gomock.Controller
		mockStore          *storage.MockStorage
		mockClient         *activities.MockClient
		mockScheduleClient *activities.MockScheduleClient
		mockHandle         *activities.MockScheduleHandle
		rs                 *RecreateSchedules
		stackName          string
		connectorID        models.ConnectorID
		connectorConfig    json.RawMessage
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
		stackName = "test-stack"
		mockStore = storage.NewMockStorage(ctrl)
		mockClient = activities.NewMockClient(ctrl)
		mockScheduleClient = activities.NewMockScheduleClient(ctrl)
		mockHandle = activities.NewMockScheduleHandle(ctrl)

		rs = NewRecreateSchedules(logger, mockClient, mockStore, stackName)

		connectorID = models.ConnectorID{Reference: uuid.New(), Provider: "stripe"}
		connectorConfig = json.RawMessage(`{"name": "test-connector", "pollingPeriod": "5m"}`)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Run", func() {
		It("should succeed with no connectors", func(ctx SpecContext) {
			mockStore.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{Data: []models.Connector{}},
				nil,
			)

			err := rs.Run(ctx)
			Expect(err).To(BeNil())
		})

		It("should return error when listing connectors fails", func(ctx SpecContext) {
			mockStore.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				nil,
				fmt.Errorf("db connection error"),
			)

			err := rs.Run(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to list connectors"))
		})

		It("should skip connectors scheduled for deletion", func(ctx SpecContext) {
			mockStore.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{
					Data: []models.Connector{
						{
							ConnectorBase: models.ConnectorBase{
								ID:       connectorID,
								Name:     "deleted-connector",
								Provider: "stripe",
							},
							ScheduledForDeletion: true,
							Config:               connectorConfig,
						},
					},
				},
				nil,
			)

			// No call to ConnectorTasksTreeGet expected
			err := rs.Run(ctx)
			Expect(err).To(BeNil())
		})

		It("should skip connector with no task tree", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     "test-connector",
					Provider: "stripe",
				},
				Config: connectorConfig,
			}

			mockStore.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{Data: []models.Connector{connector}},
				nil,
			)
			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(nil, nil)

			err := rs.Run(ctx)
			Expect(err).To(BeNil())
		})

		It("should continue processing when one connector fails", func(ctx SpecContext) {
			connectorID2 := models.ConnectorID{Reference: uuid.New(), Provider: "adyen"}
			connector1 := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     "failing-connector",
					Provider: "stripe",
				},
				Config: connectorConfig,
			}
			connector2 := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID2,
					Name:     "ok-connector",
					Provider: "adyen",
				},
				Config: connectorConfig,
			}

			mockStore.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{Data: []models.Connector{connector1, connector2}},
				nil,
			)

			// First connector: task tree fetch fails
			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(
				nil, fmt.Errorf("task tree error"),
			)

			// Second connector: no task tree
			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID2).Return(nil, nil)

			// Should not return error - continues with other connectors
			err := rs.Run(ctx)
			Expect(err).To(BeNil())
		})
	})

	Context("recreateConnectorSchedules", func() {
		It("should create schedule for periodic fetch accounts task", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     "test-connector",
					Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{
					TaskType:     models.TASK_FETCH_ACCOUNTS,
					Periodically: true,
					NextTasks:    []models.ConnectorTaskTree{},
				},
			}

			expectedScheduleID := fmt.Sprintf("%s-%s-%s", stackName, connectorID.String(), models.CAPABILITY_FETCH_ACCOUNTS.String())

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.ID).To(Equal(expectedScheduleID))
				Expect(opts.TriggerImmediately).To(BeTrue())
				Expect(opts.Overlap).To(Equal(enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE))
				Expect(opts.Spec.Intervals).To(HaveLen(1))
				Expect(opts.Spec.Intervals[0].Every).To(Equal(5 * time.Minute))

				action, ok := opts.Action.(*client.ScheduleWorkflowAction)
				Expect(ok).To(BeTrue())
				Expect(action.Workflow).To(Equal(workflow.RunFetchNextAccounts))
				Expect(action.TaskQueue).To(Equal(fmt.Sprintf("%s-default", stackName)))
			}).Return(mockHandle, nil)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})

		It("should create schedules for all periodic task types", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     "test-connector",
					Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: true},
				{TaskType: models.TASK_FETCH_PAYMENTS, Periodically: true},
				{TaskType: models.TASK_FETCH_BALANCES, Periodically: true},
				{TaskType: models.TASK_FETCH_EXTERNAL_ACCOUNTS, Periodically: true},
				{TaskType: models.TASK_FETCH_OTHERS, Name: "custom", Periodically: true},
			}

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()

			var mu sync.Mutex
			createdSchedules := make(map[string]string)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, opts client.ScheduleOptions) (client.ScheduleHandle, error) {
				action := opts.Action.(*client.ScheduleWorkflowAction)
				mu.Lock()
				createdSchedules[opts.ID] = action.Workflow.(string)
				mu.Unlock()
				return mockHandle, nil
			}).Times(5)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
			Expect(createdSchedules).To(HaveLen(5))
		})

		It("should skip non-periodic tasks", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     "test-connector",
					Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{TaskType: models.TASK_CREATE_WEBHOOKS, Periodically: false},
				{TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: false},
			}

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			// No schedule creation expected

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})

		It("should use default polling period when config has zero value", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID:       connectorID,
					Name:     "test-connector",
					Provider: "stripe",
				},
				Config: json.RawMessage(`{"name": "test"}`),
			}

			taskTree := models.ConnectorTasksTree{
				{TaskType: models.TASK_FETCH_PAYMENTS, Periodically: true},
			}

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.Spec.Intervals[0].Every).To(Equal(30 * time.Minute))
				Expect(opts.Spec.Jitter).To(Equal(5 * time.Minute))
			}).Return(mockHandle, nil)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})
	})

	Context("createSchedule idempotency", func() {
		It("should return nil when schedule already exists (AlreadyExists)", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(
				nil, serviceerror.NewAlreadyExists("already exists"),
			)

			err := rs.createSchedule(ctx, "test-schedule", "FetchAccounts", 5*time.Minute, "queue", nil, nil)
			Expect(err).To(BeNil())
		})

		It("should return nil when workflow already started (WorkflowExecutionAlreadyStarted)", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(
				nil, serviceerror.NewWorkflowExecutionAlreadyStarted("wf already started", "", ""),
			)

			err := rs.createSchedule(ctx, "test-schedule", "FetchAccounts", 5*time.Minute, "queue", nil, nil)
			Expect(err).To(BeNil())
		})

		It("should return nil when SDK reports ErrScheduleAlreadyRunning", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(
				nil, sdktemporal.ErrScheduleAlreadyRunning,
			)

			err := rs.createSchedule(ctx, "test-schedule", "FetchAccounts", 5*time.Minute, "queue", nil, nil)
			Expect(err).To(BeNil())
		})

		It("should return error for unexpected failures", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("temporal unavailable"),
			)

			err := rs.createSchedule(ctx, "test-schedule", "FetchAccounts", 5*time.Minute, "queue", nil, nil)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("temporal unavailable"))
		})
	})

	Context("buildScheduleParams", func() {
		It("should map all task types correctly", func() {
			tests := []struct {
				taskType     models.TaskType
				expectedWf   string
				expectedCap  models.Capability
			}{
				{models.TASK_FETCH_ACCOUNTS, workflow.RunFetchNextAccounts, models.CAPABILITY_FETCH_ACCOUNTS},
				{models.TASK_FETCH_BALANCES, workflow.RunFetchNextBalances, models.CAPABILITY_FETCH_BALANCES},
				{models.TASK_FETCH_EXTERNAL_ACCOUNTS, workflow.RunFetchNextExternalAccounts, models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS},
				{models.TASK_FETCH_PAYMENTS, workflow.RunFetchNextPayments, models.CAPABILITY_FETCH_PAYMENTS},
				{models.TASK_FETCH_OTHERS, workflow.RunFetchNextOthers, models.CAPABILITY_FETCH_OTHERS},
				{models.TASK_CREATE_WEBHOOKS, workflow.RunCreateWebhooks, models.CAPABILITY_CREATE_WEBHOOKS},
			}

			for _, tt := range tests {
				task := models.ConnectorTaskTree{TaskType: tt.taskType, Name: "test"}
				wf, cap, req := rs.buildScheduleParams(task, connectorID, nil)
				Expect(wf).To(Equal(tt.expectedWf), "workflow mismatch for task type %d", tt.taskType)
				Expect(cap).To(Equal(tt.expectedCap), "capability mismatch for task type %d", tt.taskType)
				Expect(req).NotTo(BeNil())
			}
		})

		It("should return empty for unknown task type", func() {
			task := models.ConnectorTaskTree{TaskType: 99}
			wf, _, req := rs.buildScheduleParams(task, connectorID, nil)
			Expect(wf).To(BeEmpty())
			Expect(req).To(BeNil())
		})
	})

	Context("jitter calculation", func() {
		It("should cap jitter at 5 minutes for long polling periods", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.Spec.Jitter).To(Equal(5 * time.Minute))
			}).Return(mockHandle, nil)

			err := rs.createSchedule(ctx, "test", "wf", 30*time.Minute, "queue", nil, nil)
			Expect(err).To(BeNil())
		})

		It("should use half of polling period when under 10 minutes", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient)
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.Spec.Jitter).To(Equal(3 * time.Minute))
			}).Return(mockHandle, nil)

			err := rs.createSchedule(ctx, "test", "wf", 6*time.Minute, "queue", nil, nil)
			Expect(err).To(BeNil())
		})
	})
})
