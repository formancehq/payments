package dummypay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
	"github.com/numary/payments/pkg/bridge/task"
)

const (
	taskKeyGenerateFiles = "generate-files"
	asset                = "DUMMYCOIN"
	generatedFilePrefix  = "dummypay-generated-file"
)

// newTaskGenerateFiles returns a new task descriptor for the task that generates files.
func newTaskGenerateFiles() TaskDescriptor {
	return TaskDescriptor{
		Key: taskKeyGenerateFiles,
	}
}

// taskGenerateFiles generates payment files to a given directory.
func taskGenerateFiles(config Config) task.Task {
	return func(ctx context.Context, logger sharedlogging.Logger,
		scheduler task.Scheduler[TaskDescriptor]) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(config.FileGenerationPeriod):
				key := fmt.Sprintf("%s-%d", generatedFilePrefix, time.Now().UnixNano())
				fileKey := fmt.Sprintf("%s/%s.json", config.Directory, key)

				var paymentObj payment

				// Generate a random payment.
				paymentObj.Reference = key
				paymentObj.Type = generateRandomType()
				paymentObj.Status = generateRandomStatus()
				paymentObj.InitialAmount = int64(generateRandomNumber())
				paymentObj.Asset = asset

				file, err := os.Create(fileKey)
				if err != nil {
					return fmt.Errorf("failed to create file: %w", err)
				}

				// Encode the payment object as JSON to a new file.
				err = json.NewEncoder(file).Encode(&paymentObj)
				if err != nil {
					// Close the file before returning.
					if fileCloseErr := file.Close(); fileCloseErr != nil {
						return fmt.Errorf("failed to close file: %w", fileCloseErr)
					}

					return fmt.Errorf("failed to encode json into file: %w", err)
				}

				// Close the file.
				if err = file.Close(); err != nil {
					return fmt.Errorf("failed to close file: %w", err)
				}
			}
		}
	}
}

// nMax is the maximum number that can be generated
// with the minimum being 0.
const nMax = 10000

// generateRandomNumber generates a random number between 0 and nMax.
func generateRandomNumber() int {
	rand.Seed(time.Now().UnixNano())

	value := rand.Intn(nMax)

	return value
}

// generateRandomType generates a random payment type.
func generateRandomType() string {
	// 50% chance.
	paymentType := payments.TypePayIn

	// 50% chance.
	if generateRandomNumber() > nMax/2 {
		paymentType = payments.TypePayout
	}

	return paymentType
}

// generateRandomStatus generates a random payment status.
func generateRandomStatus() payments.Status {
	// ~50% chance.
	paymentStatus := payments.StatusSucceeded

	n := generateRandomNumber()

	switch {
	case n < nMax/4: // 25% chance
		paymentStatus = payments.StatusPending
	case n < nMax/3: // ~9% chance
		paymentStatus = payments.StatusFailed
	case n < nMax/2: // ~16% chance
		paymentStatus = payments.StatusCancelled
	}

	return paymentStatus
}
