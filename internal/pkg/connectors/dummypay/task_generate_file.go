package dummypay

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/numary/payments/internal/pkg/payments"
	"github.com/numary/payments/internal/pkg/task"
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
func taskGenerateFiles(config Config, fs fs) task.Task {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(config.FileGenerationPeriod):
				err := generateFile(config, fs)
				if err != nil {
					return err
				}
			}
		}
	}
}

func generateFile(config Config, fs fs) error {
	key := fmt.Sprintf("%s-%d", generatedFilePrefix, time.Now().UnixNano())
	fileKey := fmt.Sprintf("%s/%s.json", config.Directory, key)

	var paymentObj payment

	// Generate a random payment.
	paymentObj.Reference = key
	paymentObj.Type = generateRandomType()
	paymentObj.Status = generateRandomStatus()
	paymentObj.InitialAmount = int64(generateRandomNumber())
	paymentObj.Asset = asset
	paymentObj.Scheme = generateRandomScheme()

	file, err := fs.Create(fileKey)
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

	return nil
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

// generateRandomScheme generates a random payment scheme.
func generateRandomScheme() payments.Scheme {
	n := generateRandomNumber() / 1000

	paymentScheme := payments.SchemeCardMasterCard

	availableSchemes := []payments.Scheme{
		payments.SchemeCardMasterCard,
		payments.SchemeCardVisa,
		payments.SchemeCardDiscover,
		payments.SchemeCardJCB,
		payments.SchemeCardUnionPay,
		payments.SchemeCardAmex,
		payments.SchemeCardDiners,
		payments.SchemeSepaDebit,
		payments.SchemeSepaCredit,
		payments.SchemeApplePay,
		payments.SchemeGooglePay,
		payments.SchemeA2A,
		payments.SchemeAchDebit,
		payments.SchemeAch,
		payments.SchemeRtp,
	}

	if n < len(availableSchemes) {
		paymentScheme = availableSchemes[n]
	}

	return paymentScheme
}
