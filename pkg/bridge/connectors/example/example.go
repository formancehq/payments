package example

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
	"github.com/numary/payments/pkg/core"
)

type (
	Config struct {
		Directory string
	}
	TaskDescriptor string
)

func (cfg Config) Validate() error {
	if cfg.Directory == "" {
		return errors.New("missing directory to watch")
	}
	return nil
}

var Loader = integration.NewLoaderBuilder[Config, TaskDescriptor]("example").
	WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
		return integration.NewConnectorBuilder[TaskDescriptor]().
			WithInstall(func(ctx task.ConnectorContext[TaskDescriptor]) error {
				return ctx.Scheduler().Schedule("directory", false)
			}).
			WithResolve(func(descriptor TaskDescriptor) task.Task {
				if descriptor == "directory" {
					return func(ctx context.Context, logger sharedlogging.Logger, scheduler task.Scheduler[TaskDescriptor]) error {
						for {
							select {
							case <-ctx.Done():
								return ctx.Err()
							case <-time.After(10 * time.Second): // Could be configurable using Config object
								logger.Infof("Opening directory '%s'...", config.Directory)
								dir, err := os.ReadDir(config.Directory)
								if err != nil {
									logger.Errorf("Error opening directory '%s': %s", config.Directory, err)
									continue
								}

								logger.Infof("Found %d files", len(dir))
								for _, file := range dir {
									err = scheduler.Schedule(TaskDescriptor(file.Name()), false)
									if err != nil {
										logger.Errorf("Error scheduling task '%s': %s", file.Name(), err)
										continue
									}
								}
							}
						}
					}
				}
				return func(ctx context.Context, ingester ingestion.Ingester, resolver task.StateResolver) error {
					file, err := os.Open(filepath.Join(config.Directory, string(descriptor)))
					if err != nil {
						return err
					}

					type JsonPayment struct {
						core.Data
						Reference string `json:"reference"`
						Type      string `json:"type"`
					}

					jsonPayment := &JsonPayment{}
					err = json.NewDecoder(file).Decode(jsonPayment)
					if err != nil {
						return err
					}

					return ingester.Ingest(ctx, ingestion.Batch{
						{
							Referenced: core.Referenced{
								Reference: jsonPayment.Reference,
								Type:      jsonPayment.Type,
							},
							Payment: &jsonPayment.Data,
							Forward: true,
						},
					}, struct{}{})
				}
			}).
			Build()
	}).
	Build()
