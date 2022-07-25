package modulr

import (
	"context"
	"errors"
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
)

type Config struct {
	Credentials Credentials `json:"credentials" bson:"credentials"`
}

func (cfg Config) Validate() error {
	fmt.Println("Validating Modulr config", cfg)
	if cfg.Credentials.APIKey == "" {
		return errors.New("missing API key")
	}

	if cfg.Credentials.APISecret == "" {
		return errors.New("missing API secret")
	}

	return nil
}

type TaskDescriptor struct {
	Name      string
	AccountID string
}

func NewLoader() integration.Loader[Config, TaskDescriptor] {
	loader := integration.NewLoaderBuilder[Config, TaskDescriptor]("modulr").
		WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDescriptor] {
			return integration.NewConnectorBuilder[TaskDescriptor]().
				WithInstall(func(ctx task.ConnectorContext[TaskDescriptor]) error {
					ctx.Scheduler().Schedule(TaskDescriptor{
						Name: "fetch-accounts",
					}, false)
					return nil
				}).
				WithResolve(func(desc TaskDescriptor) task.Task {
					if desc.Name == "fetch-transactions" {
						return func(
							ctx context.Context,
						) error {
							client := NewModulrClient(config.Credentials)

							fmt.Println("Fetching transactions for account", desc.AccountID)

							transactions, err := client.GetTransactions(desc.AccountID)
							if err != nil {
								return err
							}

							for _, transaction := range transactions {
								fmt.Println(transaction)
							}

							return nil
						}
					}
					return func(
						ctx context.Context,
						scheduler task.Scheduler[TaskDescriptor],
					) error {
						logger.Info("fetch-accounts")

						client := NewModulrClient(config.Credentials)

						accounts, err := client.GetAccounts()

						if err != nil {
							return err
						}

						for _, account := range accounts {
							logger.Infof("scheduling fetch-transactions: %s", account.ID)

							err := scheduler.Schedule(TaskDescriptor{
								Name:      "fetch-transactions",
								AccountID: account.ID,
							}, false)

							if err != nil {
								return err
							}
						}

						return nil
					}
				}).
				Build()
		}).
		Build()

	return loader
}
