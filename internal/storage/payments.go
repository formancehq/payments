package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	internalTime "github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
)

type payment struct {
	bun.BaseModel `bun:"table:payments"`

	// Mandatory fields
	ID            models.PaymentID     `bun:"id,pk,type:character varying,notnull"`
	ConnectorID   models.ConnectorID   `bun:"connector_id,type:character varying,notnull"`
	Reference     string               `bun:"reference,type:text,notnull"`
	CreatedAt     internalTime.Time    `bun:"created_at,type:timestamp without time zone,notnull"`
	Type          models.PaymentType   `bun:"type,type:text,notnull"`
	InitialAmount *big.Int             `bun:"initial_amount,type:numeric,notnull"`
	Amount        *big.Int             `bun:"amount,type:numeric,notnull"`
	Asset         string               `bun:"asset,type:text,notnull"`
	Scheme        models.PaymentScheme `bun:"scheme,type:text,notnull"`

	// Scan only fields
	Status models.PaymentStatus `bun:"status,type:text,notnull,scanonly"`

	// Optional fields
	// c.f.: https://bun.uptrace.dev/guide/models.html#nulls
	SourceAccountID         *models.AccountID `bun:"source_account_id,type:character varying,nullzero"`
	DestinationAccountID    *models.AccountID `bun:"destination_account_id,type:character varying,nullzero"`
	PsuID                   *uuid.UUID        `bun:"psu_id,type:uuid,nullzero"`
	OpenBankingConnectionID *string           `bun:"open_banking_connection_id,type:character varying,nullzero"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

type paymentAdjustment struct {
	bun.BaseModel `bun:"table:payment_adjustments"`

	// Mandatory fields
	ID        models.PaymentAdjustmentID `bun:"id,pk,type:character varying,notnull"`
	PaymentID models.PaymentID           `bun:"payment_id,type:character varying,notnull"`
	Reference string                     `bun:"reference,type:text,notnull"`
	CreatedAt internalTime.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
	Status    models.PaymentStatus       `bun:"status,type:text,notnull"`
	Raw       json.RawMessage            `bun:"raw,type:json,notnull"`

	// Optional fields
	// c.f.: https://bun.uptrace.dev/guide/models.html#nulls
	Amount *big.Int `bun:"amount,type:numeric,nullzero"`
	Asset  *string  `bun:"asset,type:text,nullzero"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`
}

func (s *store) PaymentsUpsert(ctx context.Context, payments []models.Payment) error {
	paymentsToInsert := make([]payment, 0, len(payments))
	adjustmentsToInsert := make([]paymentAdjustment, 0)
	paymentsRefundedSeen := make(map[models.PaymentID]int)
	paymentsRefunded := make([]payment, 0)
	paymentsInitialAmountToAdjustSeen := make(map[models.PaymentID]int)
	paymentsInitialAmountToAdjust := make([]payment, 0)
	paymentsCapturedSeen := make(map[models.PaymentID]int)
	paymentsCaptured := make([]payment, 0)
	for _, p := range payments {
		paymentsToInsert = append(paymentsToInsert, fromPaymentModels(p))

		for _, a := range p.Adjustments {
			adjustmentsToInsert = append(adjustmentsToInsert, fromPaymentAdjustmentModels(a))
			switch a.Status {
			case models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT:
				if i, ok := paymentsInitialAmountToAdjustSeen[p.ID]; ok {
					paymentsInitialAmountToAdjust[i].InitialAmount = a.Amount
				} else {
					res := fromPaymentModels(p)
					res.InitialAmount = a.Amount
					paymentsInitialAmountToAdjust = append(paymentsInitialAmountToAdjust, res)
					paymentsInitialAmountToAdjustSeen[p.ID] = len(paymentsInitialAmountToAdjust) - 1
				}
			case models.PAYMENT_STATUS_REFUNDED:
				if i, ok := paymentsRefundedSeen[p.ID]; ok {
					paymentsRefunded[i].Amount.Add(paymentsRefunded[i].Amount, a.Amount)
				} else {
					res := fromPaymentModels(p)
					res.Amount = a.Amount
					paymentsRefunded = append(paymentsRefunded, res)
					paymentsRefundedSeen[p.ID] = len(paymentsRefunded) - 1
				}
			case models.PAYMENT_STATUS_CAPTURE, models.PAYMENT_STATUS_REFUND_REVERSED:
				if i, ok := paymentsCapturedSeen[p.ID]; ok {
					paymentsCaptured[i].Amount.Add(paymentsCaptured[i].Amount, a.Amount)
				} else {
					res := fromPaymentModels(p)
					res.Amount = a.Amount
					paymentsCaptured = append(paymentsCaptured, res)
					paymentsCapturedSeen[p.ID] = len(paymentsCaptured) - 1
				}
			}
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	if len(paymentsToInsert) > 0 {
		_, err = tx.NewInsert().
			Model(&paymentsToInsert).
			On("CONFLICT (id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return e("failed to insert payments", err)
		}
	}

	if len(paymentsInitialAmountToAdjust) > 0 {
		_, err = tx.NewInsert().
			Model(&paymentsInitialAmountToAdjust).
			On("CONFLICT (id) DO UPDATE").
			Set("initial_amount = EXCLUDED.initial_amount").
			Exec(ctx)
		if err != nil {
			return e("failed to update payment", err)
		}
	}

	if len(paymentsCaptured) > 0 {
		_, err = tx.NewInsert().
			Model(&paymentsCaptured).
			On("CONFLICT (id) DO UPDATE").
			Set("amount = payment.amount + EXCLUDED.amount").
			Exec(ctx)
		if err != nil {
			return e("failed to update payment", err)
		}
	}

	if len(paymentsRefunded) > 0 {
		_, err = tx.NewInsert().
			Model(&paymentsRefunded).
			On("CONFLICT (id) DO UPDATE").
			Set("amount = payment.amount - EXCLUDED.amount").
			Exec(ctx)
		if err != nil {
			return e("failed to update payment", err)
		}
	}

	// Track which adjustments were actually inserted (to create outbox events only for new ones)
	var insertedAdjustments []paymentAdjustment
	if len(adjustmentsToInsert) > 0 {
		err = tx.NewInsert().
			Model(&adjustmentsToInsert).
			On("CONFLICT (id) DO NOTHING").
			Returning("*").
			Scan(ctx, &insertedAdjustments)
		if err != nil {
			return e("failed to insert adjustments", err)
		}
	}

	// Create outbox events for each inserted adjustment
	if len(insertedAdjustments) > 0 {
		// Build a map of payment ID to payment model for easy lookup
		paymentMap := make(map[models.PaymentID]models.Payment)
		for _, p := range payments {
			paymentMap[p.ID] = p
		}

		outboxEvents := make([]models.OutboxEvent, 0, len(insertedAdjustments))
		for _, adj := range insertedAdjustments {
			payment, ok := paymentMap[adj.PaymentID]
			if !ok {
				// This shouldn't happen, but skip if payment not found
				continue
			}

			// Create the event payload matching EventsSendPayment format
			payload := map[string]interface{}{
				"id":            payment.ID.String(),
				"reference":     payment.Reference,
				"type":          payment.Type.String(),
				"status":        adj.Status.String(),
				"initialAmount": payment.InitialAmount.String(),
				"amount":        payment.Amount.String(),
				"scheme":        payment.Scheme.String(),
				"asset":         payment.Asset,
				"createdAt":     payment.CreatedAt,
				"connectorID":   payment.ConnectorID.String(),
				"provider":      models.ToV3Provider(payment.ConnectorID.Provider),
				"rawData":       adj.Raw,
				"metadata":      payment.Metadata,
			}

			sourceAccountID := ""
			if payment.SourceAccountID != nil {
				sourceAccountID = payment.SourceAccountID.String()
			}
			payload["sourceAccountID"] = sourceAccountID

			destinationAccountID := ""
			if payment.DestinationAccountID != nil {
				destinationAccountID = payment.DestinationAccountID.String()
			}
			payload["destinationAccountID"] = destinationAccountID

			var payloadBytes []byte
			payloadBytes, err = json.Marshal(payload)
			if err != nil {
				return fmt.Errorf("failed to marshal payment event payload: %w", err)
			}

			// Convert adjustment back to model to get idempotency key
			adjustmentModel := toPaymentAdjustmentModels(adj)
			outboxEvent := models.OutboxEvent{
				EventType:      models.OUTBOX_EVENT_PAYMENT_SAVED,
				EntityID:       payment.ID.String(),
				Payload:        payloadBytes,
				CreatedAt:      time.Now().UTC(),
				Status:         models.OUTBOX_STATUS_PENDING,
				ConnectorID:    &payment.ConnectorID,
				IdempotencyKey: adjustmentModel.IdempotencyKey(),
			}

			outboxEvents = append(outboxEvents, outboxEvent)
		}

		// Insert outbox events in the same transaction
		if len(outboxEvents) > 0 {
			if err = s.OutboxEventsInsert(ctx, tx, outboxEvents); err != nil {
				return err
			}
		}
	}

	return e("failed to commit transactions", tx.Commit())
}

func (s *store) PaymentsUpdateMetadata(ctx context.Context, id models.PaymentID, metadata map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("update payment metadata", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	var payment payment
	err = tx.NewSelect().
		Model(&payment).
		Column("id", "metadata").
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return e("update payment metadata", err)
	}

	if payment.Metadata == nil {
		payment.Metadata = make(map[string]string)
	}

	for k, v := range metadata {
		payment.Metadata[k] = v
	}

	_, err = tx.NewUpdate().
		Model(&payment).
		Column("metadata").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return e("update payment metadata", err)
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) PaymentsGet(ctx context.Context, id models.PaymentID) (*models.Payment, error) {
	var payment payment

	err := s.db.NewSelect().
		Model(&payment).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment", err)
	}

	var ajs []paymentAdjustment
	err = s.db.NewSelect().
		Model(&ajs).
		Where("payment_id = ?", id).
		Order("created_at DESC", "sort_id DESC").
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment adjustments", err)
	}

	adjustments := make([]models.PaymentAdjustment, 0, len(ajs))
	for _, a := range ajs {
		adjustments = append(adjustments, toPaymentAdjustmentModels(a))
	}

	status := models.PAYMENT_STATUS_PENDING
	if len(adjustments) > 0 {
		// This list is ordered by created_at DESC, so the first element is the
		// last adjustment, and we want the last status.
		status = adjustments[0].Status
	}
	res := toPaymentModels(payment, status)
	res.Adjustments = adjustments
	return &res, nil
}

func (s *store) PaymentsGetByReference(ctx context.Context, reference string, connectorID models.ConnectorID) (*models.Payment, error) {
	var payment payment

	err := s.db.NewSelect().
		Model(&payment).
		Where("reference = ?", reference).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment", err)
	}

	var ajs []paymentAdjustment
	err = s.db.NewSelect().
		Model(&ajs).
		Where("payment_id = ?", payment.ID).
		Order("created_at DESC", "sort_id DESC").
		Scan(ctx)
	if err != nil {
		return nil, e("failed to get payment adjustments", err)
	}

	adjustments := make([]models.PaymentAdjustment, 0, len(ajs))
	for _, a := range ajs {
		adjustments = append(adjustments, toPaymentAdjustmentModels(a))
	}

	status := models.PAYMENT_STATUS_PENDING
	if len(adjustments) > 0 {
		// This list is ordered by created_at DESC, so the first element is the
		// last adjustment, and we want the last status.
		status = adjustments[0].Status
	}
	res := toPaymentModels(payment, status)
	res.Adjustments = adjustments
	return &res, nil
}

func (s *store) PaymentsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*payment)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)

	return e("failed to delete payments", err)
}

func (s *store) PaymentsDelete(ctx context.Context, id models.PaymentID) error {
	_, err := s.db.NewDelete().
		Model((*payment)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return e("failed to delete payment", err)
}

// PaymentsDeleteFromReference TODO this deletion method is the only one emitting outbox events.
// Using the outbox pattern makes this obvious, but others flows did not either before that pattern was set up.
func (s *store) PaymentsDeleteFromReference(ctx context.Context, reference string, connectorID models.ConnectorID) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	// Get the payment before deleting it (to create outbox event)
	var p payment
	err = tx.NewSelect().
		Model(&p).
		Where("reference = ?", reference).
		Where("connector_id = ?", connectorID).
		Scan(ctx)
	if err != nil {
		pErr := e("failed to get payment", err)
		if errors.Is(pErr, ErrNotFound) {
			// Payment doesn't exist, nothing to delete or create event for
			if commitErr := tx.Commit(); commitErr != nil {
				return fmt.Errorf("failed to commit transaction: %w", commitErr)
			}
			return nil
		}
		return pErr
	}

	// Delete the payment
	_, err = tx.NewDelete().
		Model((*payment)(nil)).
		Where("reference = ?", reference).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("failed to delete payment", err)
	}

	// Create outbox event for deleted payment
	paymentID := p.ID
	payload := map[string]interface{}{
		"id": paymentID.String(),
	}

	var payloadBytes []byte
	payloadBytes, err = json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payment deleted event payload: %w", err)
	}

	outboxEvent := models.OutboxEvent{
		EventType:      models.OUTBOX_EVENT_PAYMENT_DELETED,
		EntityID:       paymentID.String(),
		Payload:        payloadBytes,
		CreatedAt:      time.Now().UTC(),
		Status:         models.OUTBOX_STATUS_PENDING,
		ConnectorID:    &connectorID,
		IdempotencyKey: fmt.Sprintf("delete:%s", paymentID.String()),
	}

	if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
		return err
	}

	return e("failed to commit transaction", tx.Commit())
}

func (s *store) PaymentsDeleteFromAccountID(ctx context.Context, accountID models.AccountID) error {
	_, err := s.db.NewDelete().
		Model((*payment)(nil)).
		Where("source_account_id = ? OR destination_account_id = ?", accountID, accountID).
		Exec(ctx)

	return e("failed to delete payments", err)
}

func (s *store) PaymentsDeleteFromPSUID(ctx context.Context, psuID uuid.UUID) error {
	_, err := s.db.NewDelete().
		Model((*payment)(nil)).
		Where("psu_id = ?", psuID).
		Exec(ctx)

	return e("failed to delete payments", err)
}

func (s *store) PaymentsDeleteFromConnectorIDAndPSUID(ctx context.Context, connectorID models.ConnectorID, psuID uuid.UUID) error {
	_, err := s.db.NewDelete().
		Model((*payment)(nil)).
		Where("connector_id = ?", connectorID).
		Where("psu_id = ?", psuID).
		Exec(ctx)

	return e("failed to delete payments", err)
}

func (s *store) PaymentsDeleteFromOpenBankingConnectionID(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, openBankingConnectionID string) error {
	_, err := s.db.NewDelete().
		Model((*payment)(nil)).
		Where("psu_id = ?", psuID).
		Where("connector_id = ?", connectorID).
		Where("open_banking_connection_id = ?", openBankingConnectionID).
		Exec(ctx)

	return e("failed to delete payments", err)
}

type PaymentQuery struct{}

type ListPaymentsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentQuery]]

func NewListPaymentsQuery(opts bunpaginate.PaginatedQueryOptions[PaymentQuery]) ListPaymentsQuery {
	return ListPaymentsQuery{
		PageSize: opts.PageSize,
		Order:    bunpaginate.OrderAsc,
		Options:  opts,
	}
}

func (s *store) paymentsQueryContext(qb query.Builder) (string, []any, error) {
	where, args, err := qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "reference",
			key == "id",
			key == "connector_id",
			key == "type",
			key == "asset",
			key == "scheme",
			key == "status",
			key == "source_account_id",
			key == "destination_account_id",
			key == "psu_id",
			key == "open_banking_connection_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil

		case key == "initial_amount",
			key == "amount":
			return fmt.Sprintf("%s %s ?", key, query.DefaultComparisonOperatorsMapping[operator]), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'metadata' column can only be used with $match: %w", ErrValidation)
			}
			match := metadataRegex.FindAllStringSubmatch(key, 3)

			key := "metadata"
			return key + " @> ?", []any{map[string]any{
				match[0][1]: value,
			}}, nil
		default:
			return "", nil, fmt.Errorf("unknown key '%s' when building query: %w", key, ErrValidation)
		}
	}))

	return where, args, err
}

func (s *store) PaymentsList(ctx context.Context, q ListPaymentsQuery) (*bunpaginate.Cursor[models.Payment], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.paymentsQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	// TODO(polo): should fetch the adjustments and get the last status and amount?
	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PaymentQuery], payment](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PaymentQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			if where != "" {
				query = query.Where(where, args...)
			}

			query.Column("payment.*", "apd.status").
				Join(`join lateral (
				select status
				from payment_adjustments apd
				where payment_id = payment.id
				order by created_at desc, sort_id desc
				limit 1
			) apd on true`)

			// TODO(polo): sorter ?
			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch payments", err)
	}

	payments := make([]models.Payment, 0, len(cursor.Data))
	for _, p := range cursor.Data {
		payments = append(payments, toPaymentModels(p, p.Status))
	}

	return &bunpaginate.Cursor[models.Payment]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     payments,
	}, nil
}

func fromPaymentModels(from models.Payment) payment {
	return payment{
		ID:                      from.ID,
		ConnectorID:             from.ConnectorID,
		Reference:               from.Reference,
		CreatedAt:               internalTime.New(from.CreatedAt),
		Type:                    from.Type,
		InitialAmount:           from.InitialAmount,
		Amount:                  from.Amount,
		Asset:                   from.Asset,
		Scheme:                  from.Scheme,
		SourceAccountID:         from.SourceAccountID,
		DestinationAccountID:    from.DestinationAccountID,
		PsuID:                   from.PsuID,
		OpenBankingConnectionID: from.OpenBankingConnectionID,
		Metadata:                from.Metadata,
	}
}

func toPaymentModels(payment payment, status models.PaymentStatus) models.Payment {
	return models.Payment{
		ID:                      payment.ID,
		ConnectorID:             payment.ConnectorID,
		InitialAmount:           payment.InitialAmount,
		Reference:               payment.Reference,
		CreatedAt:               payment.CreatedAt.Time,
		Type:                    payment.Type,
		Amount:                  payment.Amount,
		Asset:                   payment.Asset,
		Scheme:                  payment.Scheme,
		Status:                  status,
		SourceAccountID:         payment.SourceAccountID,
		DestinationAccountID:    payment.DestinationAccountID,
		PsuID:                   payment.PsuID,
		OpenBankingConnectionID: payment.OpenBankingConnectionID,
		Metadata:                payment.Metadata,
	}
}

func fromPaymentAdjustmentModels(from models.PaymentAdjustment) paymentAdjustment {
	return paymentAdjustment{
		ID:        from.ID,
		PaymentID: from.ID.PaymentID,
		Reference: from.Reference,
		CreatedAt: internalTime.New(from.CreatedAt),
		Status:    from.Status,
		Amount:    from.Amount,
		Asset:     from.Asset,
		Metadata:  from.Metadata,
		Raw:       from.Raw,
	}
}

func toPaymentAdjustmentModels(from paymentAdjustment) models.PaymentAdjustment {
	return models.PaymentAdjustment{
		ID:        from.ID,
		Reference: from.Reference,
		CreatedAt: from.CreatedAt.Time,
		Status:    from.Status,
		Amount:    from.Amount,
		Asset:     from.Asset,
		Metadata:  from.Metadata,
		Raw:       from.Raw,
	}
}
