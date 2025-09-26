package testserver

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/service"
	"github.com/formancehq/payments/cmd"
	"github.com/stretchr/testify/require"
)

type Worker struct {
	configuration Configuration
	logger        Logger
	cancel        func()
	ctx           context.Context
	errorChan     chan error
	id            string
}

func NewWorker(t T, configuration Configuration, serverID string) *Worker {
	t.Helper()

	worker := &Worker{
		id:            serverID,
		logger:        t,
		configuration: configuration,
		errorChan:     make(chan error, 1),
	}
	t.Logf("Start testing worker")
	require.NoError(t, worker.Start())
	t.Cleanup(func() {
		t.Logf("Stop testing worker")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, worker.Stop(ctx))
	})

	return worker
}

//nolint:govet
func (w *Worker) Start() error {
	rootCmd := cmd.NewRootCommand()
	args := Flags("worker", w.id, w.configuration)

	w.logger.Logf("Starting worker with flags: %s", strings.Join(args, " "))
	rootCmd.SetArgs(args)
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

	go func() {
		w.errorChan <- rootCmd.ExecuteContext(ctx)
	}()

	select {
	case <-ctx.Done():
		return errors.New("unexpected context cancel before worker ready")
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
