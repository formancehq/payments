package testserver

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/service"
	"github.com/formancehq/payments/cmd"
)

type Worker struct {
	configuration Configuration
	logger        Logger
	cancel        func()
	ctx           context.Context
	errorChan     chan error
	id            string
}

func NewWorker(configuration Configuration, logger Logger) *Worker {
	return &Worker{
		configuration: configuration,
		logger:        logger,
	}
}

func (w *Worker) Start(args []string) error {
	rootCmd := cmd.NewRootCommand()
	workerArgs := []string{"run-worker"}
	workerArgs = append(workerArgs, args...)

	w.logger.Logf("Starting worker with flags: %s", strings.Join(workerArgs, " "))
	rootCmd.SetArgs(workerArgs)
	rootCmd.SilenceErrors = true
	output := w.configuration.Output
	if output == nil {
		output = io.Discard
	}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	ctx := logging.TestingContext()
	ctx = service.ContextWithLifecycle(ctx)
	ctx, cancel := context.WithCancel(ctx)

	select {
	case <-service.Ready(ctx):
	case err := <-w.errorChan:
		cancel()
		if err != nil {
			return err
		}

		return errors.New("unexpected worker stop")
	}

	w.ctx, w.cancel = ctx, cancel
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	if w.cancel == nil {
		return nil
	}
	w.cancel()
	w.cancel = nil

	// Wait app to be marked as stopped
	select {
	case <-service.Stopped(w.ctx):
	case <-ctx.Done():
		return errors.New("worker should have been stopped")
	}

	// Ensure the app has been properly shutdown
	select {
	case err := <-w.errorChan:
		return err
	case <-ctx.Done():
		return errors.New("worker should have been stopped without error")
	}
}
