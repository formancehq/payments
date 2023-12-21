package modulr

import (
	"context"
	"math"
	"math/big"
	"regexp"
	"time"

	"github.com/formancehq/payments/cmd/connectors/internal/connectors/currency"
	"github.com/formancehq/payments/cmd/connectors/internal/connectors/modulr/client"
	"github.com/formancehq/payments/cmd/connectors/internal/ingestion"
	"github.com/formancehq/payments/cmd/connectors/internal/storage"
	"github.com/formancehq/payments/cmd/connectors/internal/task"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/stack/libs/go-libs/contextutil"
	"github.com/formancehq/stack/libs/go-libs/logging"
	"github.com/pkg/errors"
)

var (
	ReferencePatternRegexp = regexp.MustCompile("[a-zA-Z0-9 ]*")
)

func taskInitiatePayment(logger logging.Logger, modulrClient *client.Client, transferID string) task.Task {
	return func(
		ctx context.Context,
		connectorID models.ConnectorID,
		ingester ingestion.Ingester,
		scheduler task.Scheduler,
		storageReader storage.Reader,
	) error {
		logger.Info("initiate payment for transfer-initiation %s", transferID)

		transferInitiationID := models.MustTransferInitiationIDFromString(transferID)
		transfer, err := getTransfer(ctx, storageReader, transferInitiationID, true)
		if err != nil {
			return err
		}

		var paymentID *models.PaymentID
		defer func() {
			if err != nil {
				ctx, cancel := contextutil.Detached(ctx)
				defer cancel()
				if err := ingester.UpdateTransferInitiationPaymentsStatus(ctx, transfer, paymentID, models.TransferInitiationStatusFailed, err.Error(), transfer.Attempts, time.Now()); err != nil {
					logger.Error("failed to update transfer initiation status: %v", err)
				}
			}
		}()

		err = ingester.UpdateTransferInitiationPaymentsStatus(ctx, transfer, paymentID, models.TransferInitiationStatusProcessing, "", transfer.Attempts, time.Now())
		if err != nil {
			return err
		}

		logger.Info("initiate payment between", transfer.SourceAccountID, " and %s", transfer.DestinationAccountID)

		if transfer.SourceAccount == nil {
			err = errors.New("no source account provided")
			return err
		}

		if transfer.SourceAccount.Type == models.AccountTypeExternal {
			err = errors.New("payin not implemented: source account must be an internal account")
			return err
		}

		var curr string
		var precision int
		curr, precision, err = currency.GetCurrencyAndPrecisionFromAsset(supportedCurrenciesWithDecimal, transfer.Asset)
		if err != nil {
			return err
		}

		amount := big.NewFloat(0).SetInt(transfer.Amount)
		amount = amount.Quo(amount, big.NewFloat(math.Pow(10, float64(precision))))

		description := ""
		if len(transfer.Description) <= 18 && ReferencePatternRegexp.MatchString(transfer.Description) {
			description = transfer.Description
		}

		var connectorPaymentID string
		var paymentType models.PaymentType
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		switch transfer.DestinationAccount.Type {
		case models.AccountTypeInternal:
			// Transfer between internal accounts
			var resp *client.TransferResponse
			resp, err = modulrClient.InitiateTransfer(ctx, &client.TransferRequest{
				SourceAccountID: transfer.SourceAccountID.Reference,
				Destination: client.Destination{
					Type: string(client.DestinationTypeAccount),
					ID:   transfer.DestinationAccountID.Reference,
				},
				Currency:          curr,
				Amount:            amount,
				Reference:         description,
				ExternalReference: description,
				PaymentDate:       time.Now().Add(24 * time.Hour).Format("2006-01-02"),
			})
			if err != nil {
				return err
			}

			connectorPaymentID = resp.ID
			paymentType = models.PaymentTypeTransfer
		case models.AccountTypeExternal:
			// Payout to an external account
			var resp *client.PayoutResponse
			resp, err = modulrClient.InitiatePayout(ctx, &client.PayoutRequest{
				SourceAccountID: transfer.SourceAccountID.Reference,
				Destination: client.Destination{
					Type: string(client.DestinationTypeBeneficiary),
					ID:   transfer.DestinationAccountID.Reference,
				},
				Currency:          curr,
				Amount:            amount,
				Reference:         description,
				ExternalReference: description,
			})
			if err != nil {
				return err
			}

			connectorPaymentID = resp.ID
			paymentType = models.PaymentTypePayOut
		}

		paymentID = &models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: connectorPaymentID,
				Type:      paymentType,
			},
			ConnectorID: connectorID,
		}
		err = ingester.AddTransferInitiationPaymentID(ctx, transfer, paymentID, time.Now())
		if err != nil {
			return err
		}

		taskDescriptor, err := models.EncodeTaskDescriptor(TaskDescriptor{
			Name:       "Update transfer initiation status",
			Key:        taskNameUpdatePaymentStatus,
			TransferID: transfer.ID.String(),
			PaymentID:  paymentID.String(),
			Attempt:    1,
		})
		if err != nil {
			return err
		}

		ctx, _ = contextutil.DetachedWithTimeout(ctx, 10*time.Second)
		err = scheduler.Schedule(ctx, taskDescriptor, models.TaskSchedulerOptions{
			ScheduleOption: models.OPTIONS_RUN_NOW,
			RestartOption:  models.OPTIONS_RESTART_IF_NOT_ACTIVE,
		})
		if err != nil && !errors.Is(err, task.ErrAlreadyScheduled) {
			return err
		}

		return nil
	}
}

func taskUpdatePaymentStatus(
	logger logging.Logger,
	modulrClient *client.Client,
	transferID string,
	pID string,
	attempt int,
) task.Task {
	return func(
		ctx context.Context,
		ingester ingestion.Ingester,
		scheduler task.Scheduler,
		storageReader storage.Reader,
	) error {
		paymentID := models.MustPaymentIDFromString(pID)
		transferInitiationID := models.MustTransferInitiationIDFromString(transferID)
		transfer, err := getTransfer(ctx, storageReader, transferInitiationID, false)
		if err != nil {
			return err
		}
		logger.Info("attempt: ", attempt, " fetching status of ", pID)

		var status string
		var resultMessage string
		switch transfer.Type {
		case models.TransferInitiationTypeTransfer:
			var resp *client.TransferResponse
			resp, err = modulrClient.GetTransfer(ctx, paymentID.Reference)
			if err != nil {
				return err
			}

			status = resp.Status
			resultMessage = resp.Message
		case models.TransferInitiationTypePayout:
			var resp *client.PayoutResponse
			resp, err = modulrClient.GetPayout(ctx, paymentID.Reference)
			if err != nil {
				return err
			}

			status = resp.Status
			resultMessage = resp.Message
		}

		switch status {
		case "SUBMITTED", "PENDING_FOR_DATE", "PENDING_FOR_FUNDS", "VALIDATED", "SCREENING_REQ":
			taskDescriptor, err := models.EncodeTaskDescriptor(TaskDescriptor{
				Name:       "Update transfer initiation status",
				Key:        taskNameUpdatePaymentStatus,
				TransferID: transfer.ID.String(),
				Attempt:    attempt + 1,
			})
			if err != nil {
				return err
			}

			err = scheduler.Schedule(ctx, taskDescriptor, models.TaskSchedulerOptions{
				ScheduleOption: models.OPTIONS_RUN_IN_DURATION,
				Duration:       2 * time.Minute,
				RestartOption:  models.OPTIONS_RESTART_IF_NOT_ACTIVE,
			})
			if err != nil && !errors.Is(err, task.ErrAlreadyScheduled) {
				return err
			}
		case "EXT_PROC", "PROCESSED", "RECONCILED":
			err = ingester.UpdateTransferInitiationPaymentsStatus(ctx, transfer, paymentID, models.TransferInitiationStatusProcessed, "", transfer.Attempts, time.Now())
			if err != nil {
				return err
			}

			return nil
		default:
			err = ingester.UpdateTransferInitiationPaymentsStatus(ctx, transfer, paymentID, models.TransferInitiationStatusFailed, resultMessage, transfer.Attempts, time.Now())
			if err != nil {
				return err
			}

			return nil
		}

		return nil
	}
}

func getTransfer(
	ctx context.Context,
	reader storage.Reader,
	transferID models.TransferInitiationID,
	expand bool,
) (*models.TransferInitiation, error) {
	transfer, err := reader.ReadTransferInitiation(ctx, transferID)
	if err != nil {
		return nil, err
	}

	if expand {
		if transfer.SourceAccountID != nil {
			sourceAccount, err := reader.GetAccount(ctx, transfer.SourceAccountID.String())
			if err != nil {
				return nil, err
			}
			transfer.SourceAccount = sourceAccount
		}

		destinationAccount, err := reader.GetAccount(ctx, transfer.DestinationAccountID.String())
		if err != nil {
			return nil, err
		}
		transfer.DestinationAccount = destinationAccount
	}

	return transfer, nil
}