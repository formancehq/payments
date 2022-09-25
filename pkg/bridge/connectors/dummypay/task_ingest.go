package dummypay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/ingestion"
	"github.com/numary/payments/pkg/bridge/task"
)

const taskKeyIngest = "ingest"

// newTaskIngest returns a new task descriptor for the ingest task.
func newTaskIngest(filePath string) TaskDescriptor {
	return TaskDescriptor{
		Key:      taskKeyIngest,
		FileName: filePath,
	}
}

// taskIngest ingests a payment file.
func taskIngest(config Config, descriptor TaskDescriptor) task.Task {
	return func(ctx context.Context, ingester ingestion.Ingester, resolver task.StateResolver) error {
		// Open the file.
		file, err := os.Open(filepath.Join(config.Directory, descriptor.FileName))
		if err != nil {
			return fmt.Errorf("failed to open file '%s': %w", descriptor.FileName, err)
		}

		defer file.Close()

		var paymentElement payment

		// Decode the JSON file.
		err = json.NewDecoder(file).Decode(&paymentElement)
		if err != nil {
			return fmt.Errorf("failed to decode file '%s': %w", descriptor.FileName, err)
		}

		ingestionPayload := ingestion.Batch{ingestion.BatchElement{
			Referenced: payments.Referenced{
				Reference: paymentElement.Reference,
				Type:      paymentElement.Type,
			},
			Payment: &paymentElement.Data,
			Forward: true,
		}}

		// Ingest the payment into the system.
		err = ingester.Ingest(ctx, ingestionPayload, struct{}{})
		if err != nil {
			return fmt.Errorf("failed to ingest file '%s': %w", descriptor.FileName, err)
		}

		return nil
	}
}
