package engine

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/temporal"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/pkg/errors"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

type WorkerPool struct {
	logger logging.Logger

	stack string

	temporalClient client.Client

	workers map[string]Worker
	storage storage.Storage
	plugins plugins.Plugins
	rwMutex sync.RWMutex

	workflows  []temporal.DefinitionSet
	activities []temporal.DefinitionSet

	options worker.Options
}

type Worker struct {
	worker worker.Worker
}

func NewWorkerPool(
	logger logging.Logger,
	stack string,
	temporalClient client.Client,
	workflows,
	activities []temporal.DefinitionSet,
	storage storage.Storage,
	plugins plugins.Plugins,
	options worker.Options,
) *WorkerPool {
	workers := &WorkerPool{
		logger:         logger,
		stack:          stack,
		temporalClient: temporalClient,
		workers:        make(map[string]Worker),
		workflows:      workflows,
		activities:     activities,
		storage:        storage,
		options:        options,
	}

	return workers
}

func (w *WorkerPool) OnStart(ctx context.Context) error {
	w.storage.ListenConnectorsChanges(ctx, storage.HandlerConnectorsChanges{
		storage.ConnectorChangesInsert: w.onInsertPlugin,
		storage.ConnectorChangesUpdate: w.onUpdatePlugin,
		storage.ConnectorChangesDelete: w.onDeletePlugin,
	})

	query := storage.NewListConnectorsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.ConnectorQuery{}).
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
			if err := w.onStartPlugin(ctx, connector); err != nil {
				return err
			}
		}

		if !connectors.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(connectors.Next, &query)
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
	return nil
}

func (w *WorkerPool) onStartPlugin(ctx context.Context, connector models.Connector) error {
	// Even if the connector is scheduled for deletion, we still need to register
	// the plugin to be able to handle the uninstallation.
	// It will be unregistered when the uninstallation is done in the workflow
	// after the deletion of the connector entry in the database.
	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return err
	}

	err := w.plugins.RegisterPlugin(connector.ID, connector.Name, config, connector.Config)
	if err != nil {
		w.logger.Errorf("failed to register plugin: %w", err)
		// We don't want to crash the pod if the plugin registration fails,
		// otherwise, the client will not be able to remove the failing
		// connector from the database because of the crashes.
		// We just log the error and continue.
		return nil
	}

	if !connector.ScheduledForDeletion {
		err = w.AddWorker(getConnectorTaskQueue(w.stack, connector.ID))
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *WorkerPool) onInsertPlugin(ctx context.Context, connectorID models.ConnectorID) error {
	connector, err := w.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		return err
	}

	config := models.DefaultConfig()
	if err := json.Unmarshal(connector.Config, &config); err != nil {
		return err
	}

	if err := w.plugins.RegisterPlugin(connector.ID, connector.Name, config, connector.Config); err != nil {
		return err
	}

	if err := w.AddWorker(getConnectorTaskQueue(w.stack, connector.ID)); err != nil {
		return err
	}

	return nil
}

func (w *WorkerPool) onUpdatePlugin(ctx context.Context, connectorID models.ConnectorID) error {
	connector, err := w.storage.ConnectorsGet(ctx, connectorID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return w.onDeletePlugin(ctx, connectorID)
		}
		return err
	}

	// Only react to scheduled for deletion changes
	if !connector.ScheduledForDeletion {
		return nil
	}

	if err := w.RemoveWorker(connectorID.String()); err != nil {
		return err
	}

	return nil
}

func (w *WorkerPool) onDeletePlugin(ctx context.Context, connectorID models.ConnectorID) error {
	if err := w.plugins.UnregisterPlugin(connectorID); err != nil {
		return err
	}

	if err := w.RemoveWorker(connectorID.String()); err != nil {
		return err
	}

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
	return w.AddWorker(getDefaultTaskQueue(w.stack))
}

// Installing a new connector lauches a new worker
// A default one is instantiated when the workers struct is created
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

// Uninstalling a connector stops the worker
func (w *WorkerPool) RemoveWorker(name string) error {
	w.rwMutex.Lock()
	defer w.rwMutex.Unlock()

	worker, ok := w.workers[name]
	if !ok {
		return nil
	}

	worker.worker.Stop()

	delete(w.workers, name)

	w.logger.Infof("worker for connector %s removed", name)

	return nil
}
