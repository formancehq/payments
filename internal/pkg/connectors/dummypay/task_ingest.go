package dummypay

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/numary/payments/internal/pkg/ingestion"
	"github.com/numary/payments/internal/pkg/payments"
	"github.com/numary/payments/internal/pkg/task"
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
func taskIngest(config Config, descriptor TaskDescriptor, fs fs) task.Task {
	return func(ctx context.Context, ingester ingestion.Ingester) error {
		ingestionPayload, err := parseIngestionPayload(config, descriptor, fs)
		if err != nil {
			return err
		}

		// Ingest the payment into the system.
		err = ingester.IngestPayments(ctx, ingestionPayload, struct{}{})
		if err != nil {
			return fmt.Errorf("failed to ingest file '%s': %w", descriptor.FileName, err)
		}

		return nil
	}
}

func parseIngestionPayload(config Config, descriptor TaskDescriptor, fs fs) (ingestion.PaymentBatch, error) {
	// Open the file.
	file, err := fs.Open(filepath.Join(config.Directory, descriptor.FileName))
	if err != nil {
		return nil, fmt.Errorf("failed to open file '%s': %w", descriptor.FileName, err)
	}

	defer file.Close()

	var paymentElement payment

	// Decode the JSON file.
	err = json.NewDecoder(file).Decode(&paymentElement)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file '%s': %w", descriptor.FileName, err)
	}

	ingestionPayload := ingestion.PaymentBatch{ingestion.PaymentBatchElement{
		Referenced: payments.Referenced{
			Reference: paymentElement.Reference,
			Type:      paymentElement.Type,
		},
		Payment: &paymentElement.Data,
		Forward: true,
	}}

	return ingestionPayload, nil
}
