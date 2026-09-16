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
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
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

// ratePlugin satisfies both models.Plugin and models.PluginWithPayoutThrottle,
// so OnStart should create a dedicated payout worker for connectors that
// return it. The rate is a field rather than a constant so the same double can
// stand in for a connector whose payoutsPerMinute config was updated.
type ratePlugin struct {
	models.Plugin
	rate float64
}

func (p *ratePlugin) PayoutsPerSecond() float64 { return p.rate }

// newTestPool builds a WorkerPool over mock storage and a mock connector
// manager. opts selects the Temporal client: the zero value for a pool whose
// workers start, an unreachable HostPort for one whose workers cannot.
func newTestPool(opts client.Options) (*engine.WorkerPool, *storage.MockStorage, *connectors.MockManager) {
	ctrl := gomock.NewController(GinkgoT())
	logger := logging.NewDefaultLogger(GinkgoWriter, false, false, false)
	// Use NewLazyClient as worker.New() requires a properly created client
	cl, err := client.NewLazyClient(opts)
	Expect(err).To(BeNil())
	store := storage.NewMockStorage(ctrl)
	manager := connectors.NewMockManager(ctrl)

	pool := engine.NewWorkerPool(
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
	return pool, store, manager
}

func testConnector() models.Connector {
	connID := models.ConnectorID{Reference: uuid.New(), Provider: "provider1"}
	return models.Connector{
		ConnectorBase: models.ConnectorBase{ID: connID, Name: "abc-connector", Provider: connID.Provider, CreatedAt: time.Now()},
		Config:        json.RawMessage(`{}`),
	}
}

// startPoolWithConnector runs OnStart over a single connector backed by the
// given plugin, asserts it succeeded, and returns the change handlers OnStart
// registered so the insert/update paths can be driven directly.
func startPoolWithConnector(
	ctx SpecContext,
	pool *engine.WorkerPool,
	store *storage.MockStorage,
	manager *connectors.MockManager,
	conn models.Connector,
	plugin models.Plugin,
) storage.HandlerConnectorsChanges {
	var handlers storage.HandlerConnectorsChanges
	store.EXPECT().ListenConnectorsChanges(gomock.Any(), gomock.Any()).
		Do(func(_ context.Context, h storage.HandlerConnectorsChanges) { handlers = h }).Return(nil)
	store.EXPECT().ConnectorsList(gomock.Any(), gomock.Any()).Return(&paginate.Cursor[models.Connector]{
		Data: []models.Connector{conn},
	}, nil)
	manager.EXPECT().Load(conn, false, false).Return("name", json.RawMessage(`{}`), nil)
	manager.EXPECT().Get(conn.ID).Return(plugin, nil)

	// OnStart must not fail the pod, even when a queue cannot be served.
	Expect(pool.OnStart(ctx)).To(BeNil())
	return handlers
}

// expectUpdate sets the expectations one connector-update notification consumes.
func expectUpdate(store *storage.MockStorage, manager *connectors.MockManager, conn models.Connector, plugin models.Plugin) {
	store.EXPECT().ConnectorsGet(gomock.Any(), conn.ID).Return(&conn, nil)
	manager.EXPECT().Load(conn, true, false).Return("name", json.RawMessage(`{}`), nil)
	manager.EXPECT().Get(conn.ID).Return(plugin, nil)
}

var _ = Describe("Worker Tests", func() {
	Context("on start", func() {
		var (
			pool    *engine.WorkerPool
			store   *storage.MockStorage
			manager *connectors.MockManager
			conns   []models.Connector
		)
		BeforeEach(func() {
			pool, store, manager = newTestPool(client.Options{})

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
			manager.EXPECT().Get(conns[0].ID).Return(&ratePlugin{rate: 10}, nil)

			err := pool.OnStart(ctx)
			Expect(err).To(BeNil())

			payoutQueue := engine.GetPayoutTaskQueue("stackname", conns[0].ID)
			Expect(pool.HasWorker(payoutQueue)).To(BeTrue())
			// The default worker must exist alongside it: payout workflows route
			// every activity except the PSP call back to that queue.
			Expect(pool.HasWorker(engine.GetDefaultTaskQueue("stackname"))).To(BeTrue())
		})

	})

	Context("on connector update", func() {
		var (
			pool     *engine.WorkerPool
			store    *storage.MockStorage
			manager  *connectors.MockManager
			handlers storage.HandlerConnectorsChanges
			conn     models.Connector
			plugin   *ratePlugin
			queue    string
		)

		BeforeEach(func(ctx SpecContext) {
			pool, store, manager = newTestPool(client.Options{})
			conn = testConnector()
			queue = engine.GetPayoutTaskQueue("stackname", conn.ID)
			plugin = &ratePlugin{rate: 1.5}
			handlers = startPoolWithConnector(ctx, pool, store, manager, conn, plugin)

			rate, running := pool.PayoutWorkerRate(queue)
			Expect(running).To(BeTrue())
			Expect(rate).To(Equal(1.5))
		})

		DescribeTable("reconciles the payout worker against the rate the plugin now reports",
			func(ctx SpecContext, newRate, want float64) {
				plugin.rate = newRate
				expectUpdate(store, manager, conn, plugin)

				Expect(handlers[storage.ConnectorChangesUpdate](ctx, conn.ID)).To(BeNil())

				rate, running := pool.PayoutWorkerRate(queue)
				Expect(running).To(BeTrue())
				Expect(rate).To(Equal(want))
			},
			Entry("rebuilds the worker at the new rate", 6.0, 6.0),
			Entry("leaves the worker alone when the rate is unchanged", 1.5, 1.5),
			// 0 means "no dedicated payout queue", but tearing the worker down
			// would strand whatever is already queued on it, so it is kept.
			Entry("keeps the worker rather than orphaning its queue at rate 0", 0.0, 1.5),
		)
	})

	Context("when a payout worker fails to start", func() {
		var (
			pool     *engine.WorkerPool
			store    *storage.MockStorage
			manager  *connectors.MockManager
			handlers storage.HandlerConnectorsChanges
			conn     models.Connector
			queue    string
		)

		BeforeEach(func(ctx SpecContext) {
			// Deliberately unreachable, so Start fails here exactly as it does on
			// a machine with no Temporal running.
			pool, store, manager = newTestPool(client.Options{HostPort: "127.0.0.1:1"})
			conn = testConnector()
			queue = engine.GetPayoutTaskQueue("stackname", conn.ID)
			handlers = startPoolWithConnector(ctx, pool, store, manager, conn, &ratePlugin{rate: 6})
		})

		It("does not abort startup, and records no worker for the queue", func() {
			// startPoolWithConnector already asserted OnStart succeeded. A worker
			// that never started must not be left in the map looking like one that
			// is serving the queue - that is what would make the next reconcile
			// skip it and leave payouts queued with no poller until a restart.
			Expect(pool.HasWorker(queue)).To(BeFalse())
		})

		It("retries the queue on a connector update even when the rate is unchanged", func(ctx SpecContext) {
			// Same rate as before: a running worker is left alone on this path, so
			// this Get being satisfied is what proves the failed queue was retried
			// rather than skipped.
			expectUpdate(store, manager, conn, &ratePlugin{rate: 6})

			Expect(handlers[storage.ConnectorChangesUpdate](ctx, conn.ID)).To(BeNil())

			// Still unreachable, so the retry fails again and still records
			// nothing, leaving the next update free to retry once more.
			Expect(pool.HasWorker(queue)).To(BeFalse())
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
