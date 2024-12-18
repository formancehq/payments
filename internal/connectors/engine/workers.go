package engine

import (
	"sync"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/temporal"
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
	rwMutex sync.RWMutex

	workflows  []temporal.DefinitionSet
	activities []temporal.DefinitionSet

	options worker.Options
}

type Worker struct {
	worker worker.Worker
}

func NewWorkerPool(logger logging.Logger, stack string, temporalClient client.Client, workflows, activities []temporal.DefinitionSet, options worker.Options) *WorkerPool {
	workers := &WorkerPool{
		logger:         logger,
		stack:          stack,
		temporalClient: temporalClient,
		workers:        make(map[string]Worker),
		workflows:      workflows,
		activities:     activities,
		options:        options,
	}

	return workers
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
