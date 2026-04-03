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
								ID: connectorID, Name: "deleted-connector", Provider: "stripe",
							},
							ScheduledForDeletion: true,
							Config:               connectorConfig,
						},
					},
				},
				nil,
			)

			err := rs.Run(ctx)
			Expect(err).To(BeNil())
		})

		It("should return error when one connector fails", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
				},
				Config: connectorConfig,
			}

			mockStore.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Connector]{Data: []models.Connector{connector}},
				nil,
			)
			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(
				nil, fmt.Errorf("task tree error"),
			)

			err := rs.Run(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("one or more connectors failed"))
		})

		It("should skip connector with no task tree", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
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
	})

	Context("recreateConnectorSchedules - root schedules", func() {
		It("should create schedule for periodic fetch accounts task", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: true, NextTasks: []models.ConnectorTaskTree{}},
			}

			expectedScheduleID := fmt.Sprintf("%s-%s-%s", stackName, connectorID.String(), models.CAPABILITY_FETCH_ACCOUNTS.String())

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()

			// Phase 1: root schedule
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.ID).To(Equal(expectedScheduleID))
				Expect(opts.TriggerImmediately).To(BeTrue())
				Expect(opts.Overlap).To(Equal(enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE))
				action, ok := opts.Action.(*client.ScheduleWorkflowAction)
				Expect(ok).To(BeTrue())
				Expect(action.Workflow).To(Equal(workflow.RunFetchNextAccounts))
			}).Return(mockHandle, nil)

			// Phase 2: no sub-schedules in DB
			mockStore.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{Data: []models.Schedule{}}, nil,
			)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})

		It("should create schedules for all periodic task types", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
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

			// Phase 2: no sub-schedules
			mockStore.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{Data: []models.Schedule{}}, nil,
			)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
			Expect(createdSchedules).To(HaveLen(5))
		})

		It("should skip non-periodic tasks", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{TaskType: models.TASK_CREATE_WEBHOOKS, Periodically: false},
				{TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: false},
			}

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			// Phase 2: no sub-schedules
			mockStore.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{Data: []models.Schedule{}}, nil,
			)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})

		It("should use default polling period when config has zero value", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
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

			// Phase 2: no sub-schedules
			mockStore.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{Data: []models.Schedule{}}, nil,
			)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})
	})

	Context("recreateConnectorSchedules - sub-schedules", func() {
		It("should recreate sub-schedule from DB schedule + account lookup", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{
					TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: true,
					NextTasks: []models.ConnectorTaskTree{
						{TaskType: models.TASK_FETCH_BALANCES, Periodically: true},
					},
				},
			}

			accountRef := "acct_123"
			subScheduleID := fmt.Sprintf("%s-%s-FETCH_BALANCES-%s", stackName, connectorID.String(), accountRef)

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()

			// Phase 1: root schedule created
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, opts client.ScheduleOptions) (client.ScheduleHandle, error) {
				return mockHandle, nil
			}).AnyTimes()

			// Phase 2: sub-schedule in DB
			mockStore.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{
					Data: []models.Schedule{
						{ID: subScheduleID, ConnectorID: connectorID},
					},
				}, nil,
			)

			// Account lookup
			mockStore.EXPECT().AccountsGet(gomock.Any(), models.AccountID{
				Reference:   accountRef,
				ConnectorID: connectorID,
			}).Return(&models.Account{
				ID:          models.AccountID{Reference: accountRef, ConnectorID: connectorID},
				ConnectorID: connectorID,
				Reference:   accountRef,
				CreatedAt:   time.Now(),
				Raw:         json.RawMessage(`{"id": "acct_123", "type": "standard"}`),
			}, nil)

			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})

		It("should skip sub-schedule when account not found", func(ctx SpecContext) {
			connector := models.Connector{
				ConnectorBase: models.ConnectorBase{
					ID: connectorID, Name: "test-connector", Provider: "stripe",
				},
				Config: connectorConfig,
			}

			taskTree := models.ConnectorTasksTree{
				{TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: true},
			}

			subScheduleID := fmt.Sprintf("%s-%s-FETCH_BALANCES-unknown_acct", stackName, connectorID.String())

			mockStore.EXPECT().ConnectorTasksTreeGet(gomock.Any(), connectorID).Return(&taskTree, nil)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()

			// Phase 1
			mockScheduleClient.EXPECT().Create(gomock.Any(), gomock.Any()).Return(mockHandle, nil)

			// Phase 2: sub-schedule in DB
			mockStore.EXPECT().SchedulesList(gomock.Any(), gomock.Any()).Return(
				&bunpaginate.Cursor[models.Schedule]{
					Data: []models.Schedule{
						{ID: subScheduleID, ConnectorID: connectorID},
					},
				}, nil,
			)

			// Account not found
			mockStore.EXPECT().AccountsGet(gomock.Any(), gomock.Any()).Return(
				nil, fmt.Errorf("not found"),
			)

			// Should succeed (skip the failed sub-schedule, no error propagated)
			err := rs.recreateConnectorSchedules(ctx, connector)
			Expect(err).To(BeNil())
		})
	})

	Context("parseScheduleID", func() {
		It("should parse root schedule ID", func() {
			prefix := fmt.Sprintf("%s-%s-", stackName, connectorID.String())
			scheduleID := fmt.Sprintf("%s-%s-FETCH_ACCOUNTS", stackName, connectorID.String())

			cap, payloadID, ok := rs.parseScheduleID(scheduleID, prefix)
			Expect(ok).To(BeTrue())
			Expect(cap).To(Equal("FETCH_ACCOUNTS"))
			Expect(payloadID).To(BeEmpty())
		})

		It("should parse sub-schedule ID with payload", func() {
			prefix := fmt.Sprintf("%s-%s-", stackName, connectorID.String())
			scheduleID := fmt.Sprintf("%s-%s-FETCH_BALANCES-acct_123", stackName, connectorID.String())

			cap, payloadID, ok := rs.parseScheduleID(scheduleID, prefix)
			Expect(ok).To(BeTrue())
			Expect(cap).To(Equal("FETCH_BALANCES"))
			Expect(payloadID).To(Equal("acct_123"))
		})

		It("should handle payload ID containing dashes", func() {
			prefix := fmt.Sprintf("%s-%s-", stackName, connectorID.String())
			scheduleID := fmt.Sprintf("%s-%s-FETCH_PAYMENTS-acct-with-dashes", stackName, connectorID.String())

			cap, payloadID, ok := rs.parseScheduleID(scheduleID, prefix)
			Expect(ok).To(BeTrue())
			Expect(cap).To(Equal("FETCH_PAYMENTS"))
			Expect(payloadID).To(Equal("acct-with-dashes"))
		})

		It("should parse FETCH_EXTERNAL_ACCOUNTS before FETCH_ACCOUNTS", func() {
			prefix := fmt.Sprintf("%s-%s-", stackName, connectorID.String())
			scheduleID := fmt.Sprintf("%s-%s-FETCH_EXTERNAL_ACCOUNTS-ext_123", stackName, connectorID.String())

			cap, payloadID, ok := rs.parseScheduleID(scheduleID, prefix)
			Expect(ok).To(BeTrue())
			Expect(cap).To(Equal("FETCH_EXTERNAL_ACCOUNTS"))
			Expect(payloadID).To(Equal("ext_123"))
		})

		It("should return false for invalid prefix", func() {
			_, _, ok := rs.parseScheduleID("wrong-prefix-FETCH_ACCOUNTS", "test-stack-connID-")
			Expect(ok).To(BeFalse())
		})

		It("should return false for unknown capability", func() {
			prefix := fmt.Sprintf("%s-%s-", stackName, connectorID.String())
			scheduleID := fmt.Sprintf("%s-%s-UNKNOWN_CAPABILITY", stackName, connectorID.String())

			_, _, ok := rs.parseScheduleID(scheduleID, prefix)
			Expect(ok).To(BeFalse())
		})
	})

	Context("findNextTasksForCapability", func() {
		It("should find next tasks at root level", func() {
			tree := models.ConnectorTasksTree{
				{
					TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: true,
					NextTasks: []models.ConnectorTaskTree{
						{TaskType: models.TASK_FETCH_BALANCES, Periodically: true},
					},
				},
			}

			nextTasks := rs.findNextTasksForCapability(tree, models.TASK_FETCH_BALANCES)
			Expect(nextTasks).To(BeNil()) // FETCH_BALANCES has no NextTasks itself
		})

		It("should find nested tasks", func() {
			subTasks := []models.ConnectorTaskTree{
				{TaskType: models.TASK_FETCH_PAYMENTS, Periodically: true},
			}
			tree := models.ConnectorTasksTree{
				{
					TaskType: models.TASK_FETCH_ACCOUNTS, Periodically: true,
					NextTasks: []models.ConnectorTaskTree{
						{TaskType: models.TASK_FETCH_BALANCES, Periodically: true, NextTasks: subTasks},
					},
				},
			}

			nextTasks := rs.findNextTasksForCapability(tree, models.TASK_FETCH_BALANCES)
			Expect(nextTasks).To(HaveLen(1))
			Expect(nextTasks[0].TaskType).To(Equal(models.TASK_FETCH_PAYMENTS))
		})
	})

	Context("buildScheduleID", func() {
		It("should include task name for FETCH_OTHERS", func() {
			id := rs.buildScheduleID(connectorID, models.CAPABILITY_FETCH_OTHERS, "custom-task", nil)
			Expect(id).To(ContainSubstring("FETCH_OTHERS-custom-task"))
		})

		It("should not include task name for other capabilities", func() {
			id := rs.buildScheduleID(connectorID, models.CAPABILITY_FETCH_ACCOUNTS, "ignored", nil)
			Expect(id).NotTo(ContainSubstring("ignored"))
			Expect(id).To(ContainSubstring("FETCH_ACCOUNTS"))
		})

		It("should include fromPayload ID when present", func() {
			fp := &workflow.FromPayload{ID: "acct_456"}
			id := rs.buildScheduleID(connectorID, models.CAPABILITY_FETCH_BALANCES, "", fp)
			Expect(id).To(ContainSubstring("FETCH_BALANCES-acct_456"))
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

		It("should return nil when workflow already started", func(ctx SpecContext) {
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
