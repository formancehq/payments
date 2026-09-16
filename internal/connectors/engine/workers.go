package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/observe/log"
	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/go-libs/v5/pkg/workflow/temporal"
	"github.com/formancehq/payments/internal/connectors"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/pkg/errors"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

type WorkerPool struct {
	logger logging.Logger

	stack string

	temporalClient client.Client

	workers map[string]Worker
	storage storage.Storage
	rwMutex sync.RWMutex

	workflows  []temporal.DefinitionSet
	activities []temporal.DefinitionSet

	connectors connectors.Manager
	options    worker.Options

	// skipScheduleCreation if true, skips creating the outbox publisher schedule
	// Useful for tests that don't have a Temporal server available
	skipScheduleCreation bool
	outboxPollingPeriod  time.Duration
	outboxCleanupPeriod  time.Duration
}

type Worker struct {
	worker worker.Worker

	// payoutsPerSecond is the rate this worker's task queue was created with,
	// so a connector update can tell whether the queue has to be rebuilt. Zero
	// on every worker that is not a payout worker.
	payoutsPerSecond float64

	// started is false when Start returned an error. The entry is still kept so
	// a failed worker does not abort connector startup, but the recorded rate
	// must not be trusted: syncPayoutWorker rebuilds an unstarted worker on the
	// next reconcile instead of skipping it because the rate already matches.
	started bool
}

func NewWorkerPool(
	logger logging.Logger,
	stack string,
	temporalClient client.Client,
	workflows,
	activities []temporal.DefinitionSet,
	storage storage.Storage,
	connectors connectors.Manager,
	options worker.Options,
	outboxPollingPeriod time.Duration,
	outboxCleanupPeriod time.Duration,
) *WorkerPool {
	workers := &WorkerPool{
		logger:              logger,
		stack:               stack,
		temporalClient:      temporalClient,
		workers:             make(map[string]Worker),
		workflows:           workflows,
		activities:          activities,
		storage:             storage,
		connectors:          connectors,
		options:             options,
		outboxPollingPeriod: outboxPollingPeriod,
		outboxCleanupPeriod: outboxCleanupPeriod,
	}
	return workers
}

func (w *WorkerPool) OnStart(ctx context.Context) error {
	if err := w.storage.ListenConnectorsChanges(ctx, storage.HandlerConnectorsChanges{
		storage.ConnectorChangesInsert: w.onInsertPlugin,
		storage.ConnectorChangesUpdate: w.onUpdatePlugin,
		storage.ConnectorChangesDelete: w.onDeletePlugin,
	}); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	query := storage.NewListConnectorsQuery(
		paginate.NewPaginatedQueryOptions(storage.ConnectorQuery{}).
			WithPageSize(100),
	)

	shouldCreateDefaultWorker := false
	for {
		connectors, err := w.storage.ConnectorsList(ctx, query)
		if err != nil {
			return err
		}

		shouldCreateDefaultWorker = shouldCreateDefaultWorker || len(connectors.Data) > 0
		for _, connector := range connectors.Data {
			if err := w.onStartPlugin(connector); err != nil {
				return err
			}
		}

		if !connectors.HasMore {
			break
		}

		err = paginate.UnmarshalCursor(connectors.Next, &query)
		if err != nil {
			return err
		}
	}

	if shouldCreateDefaultWorker {
		// If we have at least one connector, we need to create the default worker
		// to handle the possible tasks that are not related to a specific connector.
		// (ex: pools, bank accounts, uninstallation etc...)
		if err := w.AddDefaultWorker(); err != nil {
			return err
		}
	}

	// Create the outbox publisher schedule (unless explicitly skipped)
	if !w.skipScheduleCreation {
		if err := w.CreateOutboxPublisherSchedule(ctx); err != nil {
			return fmt.Errorf("failed to create outbox publisher schedule: %w", err)
		}
		if err := w.CreateOutboxCleanupSchedule(ctx); err != nil {
			return fmt.Errorf("failed to create outbox cleanup schedule: %w", err)
		}
	}

	return nil
}

func (w *WorkerPool) onStartPlugin(connector models.Connector) error {
	// skip strict polling period validation if installed by another instance
	_, _, err := w.connectors.Load(connector, false, false)
	if err != nil {
		w.logger.Errorf("failed to register plugin for connector %q: %s", connector.ID.String(), err.Error())
		// We don't want to crash the pod if the plugin registration fails,
		// otherwise, the client will not be able to remove the failing
		// connector from the database because of the crashes.
		// We just log the error and continue.
		return nil
	}

	// Even if the connector is scheduled for deletion, we still need to register
	// the plugin to be able to handle the uninstallation.
	// It will be unregistered when the uninstallation is done in the workflow
	// after the deletion of the connector entry in the database.
	if !connector.ScheduledForDeletion {
		if err = w.AddWorker(GetDefaultTaskQueue(w.stack)); err != nil {
			return err
		}

		if plugin, getErr := w.connectors.Get(connector.ID); getErr == nil {
			if throttle, ok := plugin.(models.PluginWithPayoutThrottle); ok && throttle.PayoutsPerSecond() > 0 {
				if err := w.AddPayoutWorker(GetPayoutTaskQueue(w.stack, connector.ID), throttle.PayoutsPerSecond()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (w *WorkerPool) onInsertPlugin(ctx context.Context, connectorID models.ConnectorID) error {
	w.logger.Debugf("worker got insert notification for %q", connectorID.String())
	connector, err := w.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return err
	}

	// skip strict polling period validation if installed by another instance
	_, _, err = w.connectors.Load(*connector, false, false)
	if err != nil {
		return err
	}

	if err := w.AddWorker(GetDefaultTaskQueue(w.stack)); err != nil {
		return err
	}

	// If we have at least one connector, we need to create the default worker
	// to handle the possible tasks that are not related to a specific connector.
	// (ex: pools, bank accounts, uninstallation etc...)
	if err := w.AddDefaultWorker(); err != nil {
		return err
	}

	if plugin, getErr := w.connectors.Get(connectorID); getErr == nil {
		if throttle, ok := plugin.(models.PluginWithPayoutThrottle); ok && throttle.PayoutsPerSecond() > 0 {
			if err := w.AddPayoutWorker(GetPayoutTaskQueue(w.stack, connectorID), throttle.PayoutsPerSecond()); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *WorkerPool) onUpdatePlugin(ctx context.Context, connectorID models.ConnectorID) error {
	w.logger.Debugf("worker got update notification for %q", connectorID.String())
	connector, err := w.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return w.onDeletePlugin(ctx, connectorID)
		}
		return err
	}

	// skip strict polling period validation if installed by another instance
	_, _, err = w.connectors.Load(*connector, true, false)
	if err != nil {
		w.logger.Errorf("failed to register plugin after update to connector %q: %v", connector.ID.String(), err)
		return err
	}

	if !connector.ScheduledForDeletion {
		if err := w.syncPayoutWorker(connector.ID); err != nil {
			return err
		}
	}
	return nil
}

// syncPayoutWorker brings the connector's payout task queue in line with the
// rate the freshly loaded plugin now reports. TaskQueueActivitiesPerSecond is
// fixed when the worker is built, so a changed rate means tearing the worker
// down and starting a new one on the same queue - workflows stay queued across
// the gap and the new worker picks them up.
func (w *WorkerPool) syncPayoutWorker(connectorID models.ConnectorID) error {
	plugin, err := w.connectors.Get(connectorID)
	if err != nil {
		// Not fatal: the connector stays on whatever worker it already has.
		w.logger.Errorf("cannot resolve payout rate for connector %q, leaving its payout worker as is: %v", connectorID.String(), err)
		return nil
	}

	throttle, ok := plugin.(models.PluginWithPayoutThrottle)
	if !ok {
		return nil
	}

	// A rate of 0 would orphan whatever is already on the payout queue, so the
	// existing worker is left running rather than stopped. Plugins are expected
	// to resolve an unset config back to their own default instead of 0.
	rate := throttle.PayoutsPerSecond()
	if rate <= 0 {
		return nil
	}

	name := GetPayoutTaskQueue(w.stack, connectorID)

	w.rwMutex.RLock()
	existing, running := w.workers[name]
	w.rwMutex.RUnlock()

	if running {
		switch {
		case existing.started && existing.payoutsPerSecond == rate:
			return nil
		case !existing.started:
			w.logger.Infof(
				"payout worker %s for connector %q never started, rebuilding at %.2f activities/s",
				name, connectorID.String(), rate,
			)
		default:
			w.logger.Infof(
				"payout rate for connector %q changed from %.2f to %.2f activities/s, restarting worker %s",
				connectorID.String(), existing.payoutsPerSecond, rate, name,
			)
		}
		w.stopWorker(name)
	}

	return w.AddPayoutWorker(name, rate)
}

func (w *WorkerPool) onDeletePlugin(ctx context.Context, connectorID models.ConnectorID) error {
	w.logger.Debugf("worker got delete notification for %q", connectorID.String())
	w.stopWorker(GetPayoutTaskQueue(w.stack, connectorID))
	w.connectors.Unload(connectorID)

	return nil
}

// Close is called when app is terminated
func (w *WorkerPool) Close() {
	w.rwMutex.Lock()
	defer w.rwMutex.Unlock()

	for _, worker := range w.workers {
		worker.worker.Stop()
	}
}

func (w *WorkerPool) AddDefaultWorker() error {
	return w.AddWorker(GetDefaultTaskQueue(w.stack))
}

// AddPayoutWorker creates a dedicated Temporal worker for payout and transfer
// workflows with TaskQueueActivitiesPerSecond set to payoutsPerSecond.
//
// The payout workflows themselves still run here, which is why the full
// workflow and activity sets are registered rather than a payout-only subset.
//
// TaskQueueActivitiesPerSecond must stay set rather than being replaced by a
// client-side limiter: the SDK turns off eager activity dispatch whenever it is
// non-zero, because the server does not rate limit eagerly dispatched
// activities. A client-side limiter would silently let them through.
func (w *WorkerPool) AddPayoutWorker(name string, payoutsPerSecond float64) error {
	w.rwMutex.Lock()
	defer w.rwMutex.Unlock()

	if _, ok := w.workers[name]; ok {
		return nil
	}

	opts := w.options
	opts.TaskQueueActivitiesPerSecond = payoutsPerSecond

	wkr := worker.New(w.temporalClient, name, opts)

	for _, set := range w.workflows {
		for _, wf := range set {
			wkr.RegisterWorkflowWithOptions(wf.Func, temporalworkflow.RegisterOptions{
				Name: wf.Name,
			})
		}
	}

	for _, set := range w.activities {
		for _, act := range set {
			wkr.RegisterActivityWithOptions(act.Func, activity.RegisterOptions{
				Name: act.Name,
			})
		}
	}

	// Started inline rather than in a goroutine: Start only spins up the
	// pollers and returns, and deferring it leaves a window where a worker that
	// is stopped again - which syncPayoutWorker does on every rate change - gets
	// its Start call after its Stop, which the SDK answers with a panic.
	//
	// A failure is logged rather than returned, so that one unreachable queue
	// cannot abort connector startup, and recorded on the entry rather than
	// dropped: a worker that never started must not be left looking like a
	// healthy one running at this rate, or the next reconcile would skip it and
	// the queue would sit with no poller until the process restarted.
	started := true
	if err := wkr.Start(); err != nil {
		started = false
		w.logger.Errorf("payout worker %s failed to start, will be rebuilt on the next reconcile: %v", name, err)
	} else {
		w.logger.Infof("payout worker %s started (%.2f activities/s)", name, payoutsPerSecond)
	}

	w.workers[name] = Worker{worker: wkr, payoutsPerSecond: payoutsPerSecond, started: started}

	return nil
}

func (w *WorkerPool) stopWorker(name string) {
	w.rwMutex.Lock()
	defer w.rwMutex.Unlock()

	if wkr, ok := w.workers[name]; ok {
		wkr.worker.Stop()
		delete(w.workers, name)
	}
}

// AddWorker instantiates a temporal worker
func (w *WorkerPool) AddWorker(name string) error {
	w.rwMutex.Lock()
	defer w.rwMutex.Unlock()

	if _, ok := w.workers[name]; ok {
		return nil
	}

	worker := worker.New(w.temporalClient, name, w.options)

	for _, set := range w.workflows {
		for _, workflow := range set {
			worker.RegisterWorkflowWithOptions(workflow.Func, temporalworkflow.RegisterOptions{
				Name: workflow.Name,
			})
		}
	}

	for _, set := range w.activities {
		for _, act := range set {
			worker.RegisterActivityWithOptions(act.Func, activity.RegisterOptions{
				Name: act.Name,
			})
		}
	}

	go func() {
		err := worker.Start()
		if err != nil {
			w.logger.Errorf("worker loop stopped: %v", err)
		}
	}()

	w.workers[name] = Worker{
		worker: worker,
	}

	w.logger.Infof("worker for connector %s started", name)

	return nil
}

func (w *WorkerPool) createSchedule(ctx context.Context, scheduleIDSuffix, workflowName string, interval time.Duration, errorMsg string) error {
	scheduleID := fmt.Sprintf("%s-%s", w.stack, scheduleIDSuffix)
	taskQueue := GetDefaultTaskQueue(w.stack)

	stackAttr := sdktemporal.NewSearchAttributes(
		sdktemporal.NewSearchAttributeKeyKeyword(workflow.SearchAttributeStack).ValueSet(w.stack),
	)

	// Create the schedule
	_, err := w.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{
					Every: interval,
				},
			},
		},
		Action: &client.ScheduleWorkflowAction{
			ID:                    scheduleID,
			Workflow:              workflowName,
			Args:                  []interface{}{}, // No arguments needed
			TaskQueue:             taskQueue,
			TypedSearchAttributes: stackAttr,
		},
		Overlap:               enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		TriggerImmediately:    true,
		TypedSearchAttributes: stackAttr,
	})

	if err != nil {
		// When triggering immediately or if a workflow with the same ID already exists,
		// Temporal may return either AlreadyExists (schedule exists) or
		// WorkflowExecutionAlreadyStarted (the workflow action with same ID already exists),
		// or the SDK sentinel error temporal.ErrScheduleAlreadyRunning when a schedule with the same ID
		// is already registered. All these cases should be treated as success as the desired state is achieved.
		var already *serviceerror.AlreadyExists
		var wfAlreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &wfAlreadyStarted) || errors.As(err, &already) {
			// Workflow already started with the same ID, treat as success
			return nil
		}
		if errors.Is(err, sdktemporal.ErrScheduleAlreadyRunning) {
			return nil
		}

		return fmt.Errorf("%s: %w", errorMsg, err)
	}
	return nil
}

func (w *WorkerPool) HasWorker(name string) bool {
	w.rwMutex.RLock()
	defer w.rwMutex.RUnlock()
	_, ok := w.workers[name]
	return ok
}

// PayoutWorkerHealthy reports whether a payout worker is present on the queue
// and actually started. A present-but-unstarted worker is not serving the
// queue, so callers must not read it as one that is.
func (w *WorkerPool) PayoutWorkerHealthy(name string) bool {
	w.rwMutex.RLock()
	defer w.rwMutex.RUnlock()
	wkr, ok := w.workers[name]
	return ok && wkr.started
}

// PayoutWorkerRate reports the activities-per-second a running payout worker
// was built with. The second return value is false when no worker is running on
// that queue.
func (w *WorkerPool) PayoutWorkerRate(name string) (float64, bool) {
	w.rwMutex.RLock()
	defer w.rwMutex.RUnlock()
	wkr, ok := w.workers[name]
	if !ok {
		return 0, false
	}
	return wkr.payoutsPerSecond, true
}

func (w *WorkerPool) CreateOutboxPublisherSchedule(ctx context.Context) error {
	return w.createSchedule(
		ctx,
		"outbox-publisher",
		"OutboxPublisher",
		w.outboxPollingPeriod,
		"failed to create outbox publisher schedule",
	)
}

func (w *WorkerPool) CreateOutboxCleanupSchedule(ctx context.Context) error {
	return w.createSchedule(
		ctx,
		"outbox-cleanup",
		"OutboxCleanup",
		w.outboxCleanupPeriod,
		"failed to create outbox cleanup schedule",
	)
}

// SetSkipScheduleCreation sets whether to skip creating the outbox publisher schedule.
// Useful for tests that don't have a Temporal server available.
func (w *WorkerPool) SetSkipScheduleCreation(skip bool) {
	w.skipScheduleCreation = skip
}
