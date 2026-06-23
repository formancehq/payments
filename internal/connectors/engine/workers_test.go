package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	gomock "go.uber.org/mock/gomock"
)

// basicPlugin satisfies models.Plugin (via embedding) but does not implement
// PluginWithPayoutThrottle, so OnStart should not create a payout worker for it.
type basicPlugin struct{ models.Plugin }

// payoutPlugin satisfies both models.Plugin and models.PluginWithPayoutThrottle.
// OnStart should create a dedicated payout worker for connectors that return it.
type payoutPlugin struct{ models.Plugin }

func (p *payoutPlugin) PayoutsPerSecond() float64 { return 10.0 }

var _ = Describe("Worker Tests", func() {
	Context("on start", func() {
		var (
			pool    *engine.WorkerPool
			store   *storage.MockStorage
			manager *connectors.MockManager
			conns   []models.Connector
		)
		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
			// Use NewLazyClient as worker.New() requires a properly created client
			cl, err := client.NewLazyClient(client.Options{})
			Expect(err).To(BeNil())
			store = storage.NewMockStorage(ctrl)
			manager = connectors.NewMockManager(ctrl)

			pool = engine.NewWorkerPool(
				logger,
				"stackname",
				cl,
				[]temporal.DefinitionSet{},
				[]temporal.DefinitionSet{},
				store,
				manager,
				worker.Options{},
				time.Second,
				time.Hour,
			)
			// Skip schedule creation in tests since we don't have a Temporal server
			pool.SetSkipScheduleCreation(true)

			connID1 := models.ConnectorID{Reference: uuid.New(), Provider: "provider1"}
			connID2 := models.ConnectorID{Reference: uuid.New(), Provider: "provider2"}

			conns = []models.Connector{
				{ConnectorBase: models.ConnectorBase{ID: connID1, Name: "abc-connector", Provider: connID1.Provider, CreatedAt: time.Now().Add(-time.Minute)}, Config: json.RawMessage(`{}`)},
				{ConnectorBase: models.ConnectorBase{ID: connID2, Name: "efg-connector", Provider: connID2.Provider, CreatedAt: time.Now()}, Config: json.RawMessage(`{}`)},
			}

		})

		It("should fail when listener fails", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("listener err")
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(expectedErr)
			err := pool.OnStart(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should fail when unable to fetch connectors from storage", func(ctx SpecContext) {
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(nil)

			expectedErr := fmt.Errorf("storage err")
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(nil, expectedErr)
			err := pool.OnStart(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err).To(MatchError(expectedErr))
		})

		It("should call RegisterPlugin on all connectors found", func(ctx SpecContext) {
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(nil)

			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&paginate.Cursor[models.Connector]{
				Data: conns,
			}, nil)
			manager.EXPECT().Load(conns[0], false, false).Return("name", json.RawMessage(`{}`), nil)
			manager.EXPECT().Load(conns[1], false, false).Return("name", json.RawMessage(`{}`), nil)
			// onStartPlugin checks each loaded plugin for PluginWithPayoutThrottle;
			// basicPlugin does not implement it so no payout worker should be created.
			manager.EXPECT().Get(conns[0].ID).Return(&basicPlugin{}, nil)
			manager.EXPECT().Get(conns[1].ID).Return(&basicPlugin{}, nil)
			err := pool.OnStart(ctx)
			Expect(err).To(BeNil())
		})

		It("should start a payout worker when a plugin implements PluginWithPayoutThrottle", func(ctx SpecContext) {
			store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).Return(nil)
			store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&paginate.Cursor[models.Connector]{
				Data: []models.Connector{conns[0]},
			}, nil)
			manager.EXPECT().Load(conns[0], false, false).Return("name", json.RawMessage(`{}`), nil)
			manager.EXPECT().Get(conns[0].ID).Return(&payoutPlugin{}, nil)

			err := pool.OnStart(ctx)
			Expect(err).To(BeNil())

			payoutQueue := engine.GetPayoutTaskQueue("stackname", conns[0].ID)
			Expect(pool.HasWorker(payoutQueue)).To(BeTrue())
		})
	})

	Context("createOutboxPublisherSchedule", func() {
		var (
			pool               *engine.WorkerPool
			mockClient         *activities.MockClient
			mockScheduleClient *activities.MockScheduleClient
			mockHandle         *activities.MockScheduleHandle
			stackName          string

			pollingInterval = time.Second
			cleanupInterval = time.Hour
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
			stackName = "test-stack"
			mockClient = activities.NewMockClient(ctrl)
			mockScheduleClient = activities.NewMockScheduleClient(ctrl)
			mockHandle = activities.NewMockScheduleHandle(ctrl)
			store := storage.NewMockStorage(ctrl)
			manager := connectors.NewMockManager(ctrl)
			pool = engine.NewWorkerPool(
				logger,
				stackName,
				mockClient,
				[]temporal.DefinitionSet{},
				[]temporal.DefinitionSet{},
				store,
				manager,
				worker.Options{},
				pollingInterval,
				cleanupInterval,
			)
			// Don't skip schedule creation for these tests
			pool.SetSkipScheduleCreation(false)
		})

		It("should successfully create schedule when it does not exist", func(ctx SpecContext) {
			scheduleID := fmt.Sprintf("%s-outbox-publisher", stackName)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.ID).To(Equal(scheduleID))
				Expect(opts.TriggerImmediately).To(BeTrue())
				Expect(opts.Overlap).To(Equal(enums.SCHEDULE_OVERLAP_POLICY_SKIP))
				Expect(opts.Spec.Intervals).To(HaveLen(1))
				Expect(opts.Spec.Intervals[0].Every).To(Equal(pollingInterval))
				stackKey := sdktemporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeStack)
				scheduleStack, hasScheduleStack := opts.TypedSearchAttributes.GetKeyword(stackKey)
				Expect(hasScheduleStack).To(BeTrue())
				Expect(scheduleStack).To(Equal(stackName))
				action, ok := opts.Action.(*client.ScheduleWorkflowAction)
				Expect(ok).To(BeTrue())
				Expect(action.Workflow).To(Equal("OutboxPublisher"))
				Expect(action.TaskQueue).To(Equal(fmt.Sprintf("%s-default", stackName)))
				actionStack, hasActionStack := action.TypedSearchAttributes.GetKeyword(stackKey)
				Expect(hasActionStack).To(BeTrue())
				Expect(actionStack).To(Equal(stackName))
			}).Return(mockHandle, nil)

			err := pool.CreateOutboxPublisherSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return nil when schedule already exists (AlreadyExists error)", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, serviceerror.NewAlreadyExists("already exists"))

			err := pool.CreateOutboxPublisherSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return nil when initial workflow already started (WorkflowExecutionAlreadyStarted)", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, serviceerror.NewWorkflowExecutionAlreadyStarted("wf already started", "", ""))

			err := pool.CreateOutboxPublisherSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return nil when SDK reports ErrScheduleAlreadyRunning", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, sdktemporal.ErrScheduleAlreadyRunning)

			err := pool.CreateOutboxPublisherSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return error when Create fails with non-AlreadyExists error", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("create error")
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, expectedErr)

			err := pool.CreateOutboxPublisherSchedule(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to create outbox publisher schedule"))
		})
	})

	Context("createOutboxCleanupSchedule", func() {
		var (
			pool               *engine.WorkerPool
			mockClient         *activities.MockClient
			mockScheduleClient *activities.MockScheduleClient
			mockHandle         *activities.MockScheduleHandle
			stackName          string

			pollingInterval = time.Second
			cleanupInterval = time.Hour
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
			stackName = "test-stack"
			mockClient = activities.NewMockClient(ctrl)
			mockScheduleClient = activities.NewMockScheduleClient(ctrl)
			mockHandle = activities.NewMockScheduleHandle(ctrl)
			store := storage.NewMockStorage(ctrl)
			manager := connectors.NewMockManager(ctrl)
			pool = engine.NewWorkerPool(
				logger,
				stackName,
				mockClient,
				[]temporal.DefinitionSet{},
				[]temporal.DefinitionSet{},
				store,
				manager,
				worker.Options{},
				pollingInterval,
				cleanupInterval,
			)
			// Don't skip schedule creation for these tests
			pool.SetSkipScheduleCreation(false)
		})

		It("should successfully create schedule when it does not exist", func(ctx SpecContext) {
			scheduleID := fmt.Sprintf("%s-outbox-cleanup", stackName)
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Do(func(_ context.Context, opts client.ScheduleOptions) {
				Expect(opts.ID).To(Equal(scheduleID))
				Expect(opts.TriggerImmediately).To(BeTrue())
				Expect(opts.Overlap).To(Equal(enums.SCHEDULE_OVERLAP_POLICY_SKIP))
				Expect(opts.Spec.Intervals).To(HaveLen(1))
				Expect(opts.Spec.Intervals[0].Every).To(Equal(cleanupInterval))
				stackKey := sdktemporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeStack)
				scheduleStack, hasScheduleStack := opts.TypedSearchAttributes.GetKeyword(stackKey)
				Expect(hasScheduleStack).To(BeTrue())
				Expect(scheduleStack).To(Equal(stackName))
				action, ok := opts.Action.(*client.ScheduleWorkflowAction)
				Expect(ok).To(BeTrue())
				Expect(action.Workflow).To(Equal("OutboxCleanup"))
				Expect(action.TaskQueue).To(Equal(fmt.Sprintf("%s-default", stackName)))
				actionStack, hasActionStack := action.TypedSearchAttributes.GetKeyword(stackKey)
				Expect(hasActionStack).To(BeTrue())
				Expect(actionStack).To(Equal(stackName))
			}).Return(mockHandle, nil)

			err := pool.CreateOutboxCleanupSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return nil when schedule already exists (AlreadyExists error)", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, serviceerror.NewAlreadyExists("already exists"))

			err := pool.CreateOutboxCleanupSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return nil when initial workflow already started (WorkflowExecutionAlreadyStarted)", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, serviceerror.NewWorkflowExecutionAlreadyStarted("wf already started", "", ""))

			err := pool.CreateOutboxCleanupSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return nil when SDK reports ErrScheduleAlreadyRunning", func(ctx SpecContext) {
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, sdktemporal.ErrScheduleAlreadyRunning)

			err := pool.CreateOutboxCleanupSchedule(ctx)
			Expect(err).To(BeNil())
		})

		It("should return error when Create fails with non-AlreadyExists error", func(ctx SpecContext) {
			expectedErr := fmt.Errorf("create error")
			mockClient.EXPECT().ScheduleClient().Return(mockScheduleClient).AnyTimes()
			mockScheduleClient.EXPECT().Create(ctx, gomock.Any()).Return(nil, expectedErr)

			err := pool.CreateOutboxCleanupSchedule(ctx)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to create outbox cleanup schedule"))
		})
	})
})
