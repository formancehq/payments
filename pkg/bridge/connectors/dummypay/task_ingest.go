package dummypay

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
)

const taskKeyIngest = "ingest"

func newTaskIngest(filePath string) TaskDescriptor {
	return TaskDescriptor{
		Key:      taskKeyIngest,
		FileName: filePath,
	}
}

func taskIngest(config Config, descriptor TaskDescriptor) task.Task {
	return func(ctx context.Context, ingester ingestion.Ingester, resolver task.StateResolver) error {
		file, err := os.Open(filepath.Join(config.Directory, descriptor.FileName))
		if err != nil {
			return err
		}

		type JsonPayment struct {
			payments.Data
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
				Referenced: payments.Referenced{
					Reference: jsonPayment.Reference,
					Type:      jsonPayment.Type,
				},
				Payment: &jsonPayment.Data,
				Forward: true,
			},
		}, struct{}{})
	}
}
