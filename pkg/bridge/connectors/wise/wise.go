package wise

import (
	"context"
	"fmt"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/integration"
	"github.com/numary/payments/pkg/bridge/task"
)

const (
	taskNameFetchTransfers = "fetch-transfers"
	taskNameFetchProfiles  = "fetch-profiles"
	taskNameFetchBalances  = "fetch-balances"
)

type Config struct {
	ApiKey string `json:"apiKey" yaml:"apiKey" bson:"apiKey"`
}

func (c Config) Validate() error {
	return nil
}

type TaskDefinition struct {
	Name      string `json:"name" yaml:"name" bson:"name"`
	ProfileId uint64 `json:"profileId" yaml:"profileId" bson:"profileId"`
}

func NewLoader() integration.Loader[Config, TaskDefinition] {
	loader := integration.NewLoaderBuilder[Config, TaskDefinition]("wise").
		WithLoad(func(logger sharedlogging.Logger, config Config) integration.Connector[TaskDefinition] {
			return integration.NewConnectorBuilder[TaskDefinition]().
				WithInstall(func(ctx task.ConnectorContext[TaskDefinition]) error {
					return ctx.Scheduler().Schedule(TaskDefinition{
						Name: "fetch-profiles",
					}, false)
				}).
				WithResolve(func(def TaskDefinition) task.Task {
					if def.Name == taskNameFetchProfiles {
						return func(
							ctx context.Context,
							scheduler task.Scheduler[TaskDefinition],
						) error {
							client := NewClient(config.ApiKey)

							profiles, err := client.GetProfiles()

							if err != nil {
								return err
							}

							fmt.Println(profiles)

							for _, profile := range profiles {
								logger.Infof(fmt.Sprintf("scheduling fetch-transfers: %d", profile.Id))
								scheduler.Schedule(TaskDefinition{
									Name:      fmt.Sprintf("fetch-transfers"),
									ProfileId: profile.Id,
								}, false)
							}

							return nil
						}
					}
					return func(
						ctx context.Context,
						scheduler task.Scheduler[TaskDefinition],
						ingester ingestion.Ingester,
					) error {
						client := NewClient(config.ApiKey)

						transfers, err := client.GetTransfers(&Profile{
							Id: def.ProfileId,
						})

						if err != nil {
							return err
						}

						fmt.Println(transfers)

						batch := ingestion.Batch{}

						for _, transfer := range transfers {
							logger.Info(transfer)
							batch = append(batch, ingestion.BatchElement{
								Referenced: payments.Referenced{
									Reference: fmt.Sprintf("%d", transfer.ID),
									Type:      "transfer",
								},
								Payment: &payments.Data{
									Status:        payments.StatusSucceeded,
									Scheme:        payments.SchemeOther,
									InitialAmount: int64(transfer.TargetValue * 100),
									Asset:         fmt.Sprintf("%s/2", transfer.TargetCurrency),
									Raw:           transfer,
								},
							})
						}

						return ingester.Ingest(ctx, batch, struct{}{})
					}
				}).
				Build()
		}).Build()

	return loader
}
