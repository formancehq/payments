package workflow

import (
	"errors"
	"math/big"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var ErrPaymentInitiationNotProcessed = errors.New("payment initiation not processed")

type ValidateReverse struct {
	ConnectorID models.ConnectorID
	PI          *models.PaymentInitiation
	PIReversal  *models.PaymentInitiationReversal
}

func (w Workflow) validateReverse(
	ctx workflow.Context,
	validateReverse ValidateReverse,
) error {
	// First ensure that the payment initiation was processed successfully
	err := w.validatePaymentInitiationProcessed(ctx, validateReverse)
	if err != nil {
		return err
	}

	err = w.validateReverseAmount(ctx, validateReverse)
	if err != nil {
		return err
	}

	err = w.validateOnlyReverse(ctx, validateReverse)
	if err != nil {
		return err
	}

	return nil
}

func (w Workflow) validateOnlyReverse(
	ctx workflow.Context,
	validateReverse ValidateReverse,
) error {
	now := workflow.Now(ctx)

	adj := models.PaymentInitiationAdjustment{
		ID: models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: validateReverse.PIReversal.PaymentInitiationID,
			CreatedAt:           now,
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING,
		},
		CreatedAt: now,
		Status:    models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING,
		Amount:    validateReverse.PIReversal.Amount,
		Asset:     &validateReverse.PI.Asset,
		Metadata:  validateReverse.PIReversal.Metadata,
	}

	// Second, ensure that we do not have another reverse currently being processed
	inserted, err := activities.StoragePaymentInitiationsAdjustmentsIfPredicateStore(
		infiniteRetryContext(ctx),
		adj,
		[]models.PaymentInitiationAdjustmentStatus{
			models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_PROCESSING,
		},
	)
	if err != nil {
		return err
	}

	if !inserted {
		err = errors.New("another reverse is already in progress")
		return temporal.NewNonRetryableApplicationError(err.Error(), "ANOTHER_REVERSE_IN_PROGRESS", err)
	}

	if err := w.runSendEvents(ctx, SendEvents{
		PaymentInitiationAdjustment: &adj,
	}); err != nil {
		return err
	}

	return nil
}

func (w Workflow) validatePaymentInitiationProcessed(
	ctx workflow.Context,
	validateReverse ValidateReverse,
) error {
	query := storage.NewListPaymentInitiationAdjustmentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(1).
			WithQueryBuilder(query.Match("status", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED.String())),
	)

	adjustments, err := activities.StoragePaymentInitiationAdjustmentsList(
		infiniteRetryContext(ctx),
		models.PaymentInitiationID(validateReverse.PIReversal.PaymentInitiationID),
		query,
	)
	if err != nil {
		return err
	}

	if len(adjustments.Data) > 0 {
		// Payment initiation has been processed
		return nil
	}

	return temporal.NewNonRetryableApplicationError("no adjustments found", "PAYMENT_INITIATION_NOT_PROCESSED", ErrPaymentInitiationNotProcessed)
}

func (w Workflow) validateReverseAmount(
	ctx workflow.Context,
	validateReverse ValidateReverse,
) error {
	amount := validateReverse.PI.Amount

	query := storage.NewListPaymentInitiationAdjustmentsQuery(
		bunpaginate.NewPaginatedQueryOptions(storage.PaymentInitiationAdjustmentsQuery{}).
			WithPageSize(100).
			WithQueryBuilder(query.Match("status", models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED.String())),
	)

	amountReversed := big.NewInt(0)
	for {
		adjs, err := activities.StoragePaymentInitiationAdjustmentsList(
			infiniteRetryContext(ctx),
			validateReverse.PI.ID,
			query,
		)
		if err != nil {
			return err
		}

		for _, adj := range adjs.Data {
			amountReversed.Add(amountReversed, adj.Amount)
		}

		if !adjs.HasMore {
			break
		}

		err = bunpaginate.UnmarshalCursor(adjs.Next, &query)
		if err != nil {
			return err
		}
	}

	currentAmount := new(big.Int).Sub(amount, amountReversed)
	nextAmount := new(big.Int).Sub(currentAmount, validateReverse.PIReversal.Amount)
	switch nextAmount.Sign() {
	case -1:
		// we are in the negative, we cannot reverse more than the amount
		err := errors.New("cannot reverse more than the amount")
		return temporal.NewNonRetryableApplicationError(err.Error(), "CANNOT_REVERSE_MORE_THAN_AMOUNT", err)
	default:
		return nil
	}
}

func (w Workflow) addPIReversalAdjustment(
	ctx workflow.Context,
	adjustmentID models.PaymentInitiationReversalAdjustmentID,
	err error,
	metadata map[string]string,
) error {
	adj := models.PaymentInitiationReversalAdjustment{
		ID:                          adjustmentID,
		PaymentInitiationReversalID: adjustmentID.PaymentInitiationReversalID,
		CreatedAt:                   workflow.Now(ctx),
		Status:                      adjustmentID.Status,
		Error:                       err,
		Metadata:                    metadata,
	}

	return activities.StoragePaymentInitiationReversalsAdjustmentsStore(
		infiniteRetryContext(ctx),
		adj,
	)
}
