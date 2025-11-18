package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	internalTime "github.com/formancehq/go-libs/v3/time"
	internalEvents "github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type bankAccount struct {
	bun.BaseModel `bun:"table:bank_accounts"`

	// Mandatory fields
	ID        uuid.UUID         `bun:"id,pk,type:uuid,notnull"`
	CreatedAt internalTime.Time `bun:"created_at,type:timestamp without time zone,notnull"`
	Name      string            `bun:"name,type:text,notnull"`

	// Field encrypted
	AccountNumber string `bun:"decrypted_account_number,scanonly"`
	IBAN          string `bun:"decrypted_iban,scanonly"`
	SwiftBicCode  string `bun:"decrypted_swift_bic_code,scanonly"`

	// Optional fields
	// c.f.: https://bun.uptrace.dev/guide/models.html#nulls
	Country *string `bun:"country,type:text,nullzero"`

	// Optional fields with default
	// c.f. https://bun.uptrace.dev/guide/models.html#default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	PsuID *uuid.UUID `bun:"psu_id,type:uuid,nullzero"`

	PSU             *paymentServiceUser          `bun:"rel:belongs-to,join:psu_id=id,scanonly"`
	RelatedAccounts []*bankAccountRelatedAccount `bun:"rel:has-many,join:id=bank_account_id,scanonly"`
}

func (s *store) BankAccountsUpsert(ctx context.Context, ba models.BankAccount) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("begin transaction", err)
	}

	var errTx error
	defer func() {
		rollbackOnTxError(ctx, &tx, errTx)
	}()

	toInsert := fromBankAccountModels(ba)
	// Insert or update the bank account
	res, err := tx.NewInsert().
		Model(&toInsert).
		Column("id", "created_at", "name", "country", "metadata", "psu_id").
		On("CONFLICT (id) DO NOTHING").
		Returning("id").
		Exec(ctx)
	if err != nil {
		errTx = err
		return e("insert bank account", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		errTx = err
		return e("insert bank account", err)
	}

	if rowsAffected > 0 {
		_, err = tx.NewUpdate().
			Model((*bankAccount)(nil)).
			Set("account_number = pgp_sym_encrypt(?::TEXT, ?, ?)", toInsert.AccountNumber, s.configEncryptionKey, encryptionOptions).
			Set("iban = pgp_sym_encrypt(?::TEXT, ?, ?)", toInsert.IBAN, s.configEncryptionKey, encryptionOptions).
			Set("swift_bic_code = pgp_sym_encrypt(?::TEXT, ?, ?)", toInsert.SwiftBicCode, s.configEncryptionKey, encryptionOptions).
			Where("id = ?", toInsert.ID).
			Exec(ctx)
		if err != nil {
			errTx = err
			return e("update bank account", err)
		}

		if len(toInsert.RelatedAccounts) > 0 {
			// Insert or update the related accounts
			_, err = tx.NewInsert().
				Model(&toInsert.RelatedAccounts).
				On("CONFLICT (bank_account_id, account_id) DO NOTHING").
				Exec(ctx)
			if err != nil {
				errTx = err
				return e("insert related accounts", err)
			}
		}

		// Create outbox event for new bank account
		// Convert back to model to create payload
		bankAccountModel := toBankAccountModels(toInsert)

		// Obfuscate sensitive fields before storing in outbox payload since tests read from outbox directly
		if err := bankAccountModel.Obfuscate(); err != nil {
			errTx = err
			return e("failed to obfuscate bank account for event payload", err)
		}

		// Create the event payload
		payload := prepareBankAccountEventPayload(bankAccountModel)

		var payloadBytes []byte
		payloadBytes, err = json.Marshal(&payload)
		if err != nil {
			errTx = err
			return e("failed to marshal bank account event payload", err)
		}

		outboxEvent := models.OutboxEvent{
			EventType:      events.EventTypeSavedBankAccount,
			EntityID:       bankAccountModel.ID.String(),
			Payload:        payloadBytes,
			CreatedAt:      time.Now().UTC(),
			Status:         models.OUTBOX_STATUS_PENDING,
			ConnectorID:    nil, // Bank accounts don't have connector ID
			IdempotencyKey: bankAccountModel.IdempotencyKey(),
		}

		if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
			errTx = err
			return err
		}
	} else {
		if len(toInsert.RelatedAccounts) > 0 {
			// Insert or update the related accounts even if bank account already existed
			_, err = tx.NewInsert().
				Model(&toInsert.RelatedAccounts).
				On("CONFLICT (bank_account_id, account_id) DO NOTHING").
				Exec(ctx)
			if err != nil {
				errTx = err
				return e("insert related accounts", err)
			}

			// Also create an outbox event to notify bank account update (e.g., new related account)
			bankAccountModel := toBankAccountModels(toInsert)
			// Obfuscate sensitive fields before storing in outbox payload
			if err := bankAccountModel.Obfuscate(); err != nil {
				errTx = err
				return e("failed to obfuscate bank account for event payload", err)
			}
			payload := prepareBankAccountEventPayload(bankAccountModel)

			payloadBytes, err := json.Marshal(&payload)
			if err != nil {
				errTx = err
				return e("failed to marshal bank account event payload", err)
			}
			outboxEvent := models.OutboxEvent{
				EventType:      events.EventTypeSavedBankAccount,
				EntityID:       bankAccountModel.ID.String(),
				Payload:        payloadBytes,
				CreatedAt:      time.Now().UTC(),
				Status:         models.OUTBOX_STATUS_PENDING,
				ConnectorID:    nil,
				IdempotencyKey: bankAccountModel.IdempotencyKey(),
			}
			if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
				errTx = err
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		errTx = err
		return e("commit transaction", err)
	}

	return nil
}

func (s *store) BankAccountsUpdateMetadata(ctx context.Context, id uuid.UUID, metadata map[string]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("update bank account metadata", err)
	}
	defer func() {
		rollbackOnTxError(ctx, &tx, err)
	}()

	var account bankAccount
	err = tx.NewSelect().
		Model(&account).
		Column("id", "metadata").
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return e("update bank account metadata", err)
	}

	if account.Metadata == nil {
		account.Metadata = make(map[string]string)
	}

	for k, v := range metadata {
		account.Metadata[k] = v
	}

	_, err = tx.NewUpdate().
		Model(&account).
		Column("metadata").
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return e("update bank account metadata", err)
	}

	return e("commit transaction", tx.Commit())
}

func (s *store) BankAccountsGet(ctx context.Context, id uuid.UUID, expand bool) (*models.BankAccount, error) {
	var account bankAccount
	query := s.db.NewSelect().
		Model(&account).
		Column("id", "created_at", "name", "country", "metadata", "psu_id").
		Relation("RelatedAccounts")
	if expand {
		query = query.ColumnExpr("pgp_sym_decrypt(account_number, ?, ?) AS decrypted_account_number", s.configEncryptionKey, encryptionOptions).
			ColumnExpr("pgp_sym_decrypt(iban, ?, ?) AS decrypted_iban", s.configEncryptionKey, encryptionOptions).
			ColumnExpr("pgp_sym_decrypt(swift_bic_code, ?, ?) AS decrypted_swift_bic_code", s.configEncryptionKey, encryptionOptions)
	}
	err := query.Where("id = ?", id).Scan(ctx)
	if err != nil {
		return nil, e("get bank account", err)
	}

	return pointer.For(toBankAccountModels(account)), nil
}

type BankAccountQuery struct{}

type ListBankAccountsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[BankAccountQuery]]

func NewListBankAccountsQuery(opts bunpaginate.PaginatedQueryOptions[BankAccountQuery]) ListBankAccountsQuery {
	return ListBankAccountsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) bankAccountsQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "name", key == "country", key == "id", key == "psu_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
		case metadataRegex.Match([]byte(key)):
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
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
}

func (s *store) BankAccountsList(ctx context.Context, q ListBankAccountsQuery) (*bunpaginate.Cursor[models.BankAccount], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.bankAccountsQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[BankAccountQuery], bankAccount](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[BankAccountQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			query = query.Relation("RelatedAccounts")
			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, e("failed to fetch accounts", err)
	}

	bankAccounts := make([]models.BankAccount, 0, len(cursor.Data))
	for _, a := range cursor.Data {
		bankAccounts = append(bankAccounts, toBankAccountModels(a))
	}

	return &bunpaginate.Cursor[models.BankAccount]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     bankAccounts,
	}, nil
}

type bankAccountRelatedAccount struct {
	bun.BaseModel `bun:"table:bank_accounts_related_accounts"`

	// Mandatory fields
	BankAccountID uuid.UUID          `bun:"bank_account_id,pk,type:uuid,notnull"`
	AccountID     models.AccountID   `bun:"account_id,pk,type:character varying,notnull"`
	ConnectorID   models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	CreatedAt     internalTime.Time  `bun:"created_at,type:timestamp without time zone,notnull"`
}

func (s *store) BankAccountsAddRelatedAccount(ctx context.Context, bID uuid.UUID, relatedAccount models.BankAccountRelatedAccount) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("begin transaction", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	toInsert := fromBankAccountRelatedAccountModels(relatedAccount, bID)
	_, err = tx.NewInsert().
		Model(&toInsert).
		On("CONFLICT (bank_account_id, account_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("add bank account related account", err)
	}

	// Load bank account with related accounts to build event payload
	var ba bankAccount
	err = tx.NewSelect().
		Model(&ba).
		Column("id", "created_at", "name", "country", "metadata").
		ColumnExpr("pgp_sym_decrypt(account_number, ?, ?) AS decrypted_account_number", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(iban, ?, ?) AS decrypted_iban", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(swift_bic_code, ?, ?) AS decrypted_swift_bic_code", s.configEncryptionKey, encryptionOptions).
		Relation("RelatedAccounts").
		Where("id = ?", bID).
		Scan(ctx)
	if err != nil {
		return e("load bank account for outbox event", err)
	}

	bankAccountModel := toBankAccountModels(ba)
	// Obfuscate before building payload so outbox contains masked values
	if err = bankAccountModel.Obfuscate(); err != nil {
		return e("failed to obfuscate bank account for event payload", err)
	}

	payload := prepareBankAccountEventPayload(bankAccountModel)

	var payloadBytes []byte
	payloadBytes, err = json.Marshal(&payload)
	if err != nil {
		return e("failed to marshal bank account event payload", err)
	}

	outboxEvent := models.OutboxEvent{
		EventType:      events.EventTypeSavedBankAccount,
		EntityID:       bankAccountModel.ID.String(),
		Payload:        payloadBytes,
		CreatedAt:      time.Now().UTC(),
		Status:         models.OUTBOX_STATUS_PENDING,
		ConnectorID:    nil,
		IdempotencyKey: bankAccountModel.IdempotencyKey(),
	}
	if err = s.OutboxEventsInsert(ctx, tx, []models.OutboxEvent{outboxEvent}); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return e("commit transaction", err)
	}
	return nil
}

func (s *store) BankAccountsDeleteRelatedAccountFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*bankAccountRelatedAccount)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("delete bank account related account", err)
	}

	return nil
}

func fromBankAccountModels(from models.BankAccount) bankAccount {
	ba := bankAccount{
		ID:        from.ID,
		CreatedAt: internalTime.New(from.CreatedAt),
		Name:      from.Name,
		Country:   from.Country,
		Metadata:  from.Metadata,
		PsuID:     nil,
	}

	if from.AccountNumber != nil {
		ba.AccountNumber = *from.AccountNumber
	}

	if from.IBAN != nil {
		ba.IBAN = *from.IBAN
	}

	if from.SwiftBicCode != nil {
		ba.SwiftBicCode = *from.SwiftBicCode
	}

	relatedAccounts := make([]*bankAccountRelatedAccount, 0, len(from.RelatedAccounts))
	for _, ra := range from.RelatedAccounts {
		relatedAccounts = append(relatedAccounts, pointer.For(fromBankAccountRelatedAccountModels(ra, from.ID)))
	}
	ba.RelatedAccounts = relatedAccounts

	return ba
}

func toBankAccountModels(from bankAccount) models.BankAccount {
	ba := models.BankAccount{
		ID:        from.ID,
		CreatedAt: from.CreatedAt.Time,
		Name:      from.Name,
		Country:   from.Country,
		Metadata:  from.Metadata,
	}

	if from.AccountNumber != "" {
		ba.AccountNumber = &from.AccountNumber
	}

	if from.IBAN != "" {
		ba.IBAN = &from.IBAN
	}

	if from.SwiftBicCode != "" {
		ba.SwiftBicCode = &from.SwiftBicCode
	}

	relatedAccounts := make([]models.BankAccountRelatedAccount, 0, len(from.RelatedAccounts))
	for _, ra := range from.RelatedAccounts {
		relatedAccounts = append(relatedAccounts, toBankAccountRelatedAccountModels(*ra))
	}
	ba.RelatedAccounts = relatedAccounts

	return ba
}

func fromBankAccountRelatedAccountModels(from models.BankAccountRelatedAccount, bID uuid.UUID) bankAccountRelatedAccount {
	return bankAccountRelatedAccount{
		BankAccountID: bID,
		AccountID:     from.AccountID,
		ConnectorID:   from.AccountID.ConnectorID,
		CreatedAt:     internalTime.New(from.CreatedAt),
	}
}

func toBankAccountRelatedAccountModels(from bankAccountRelatedAccount) models.BankAccountRelatedAccount {
	return models.BankAccountRelatedAccount{
		AccountID: from.AccountID,
		CreatedAt: from.CreatedAt.Time,
	}
}

func prepareBankAccountEventPayload(bankAccountModel models.BankAccount) internalEvents.BankAccountMessagePayload {
	payload := internalEvents.BankAccountMessagePayload{
		ID:        bankAccountModel.ID.String(),
		CreatedAt: bankAccountModel.CreatedAt,
		Name:      bankAccountModel.Name,
		Metadata:  bankAccountModel.Metadata,
	}

	if bankAccountModel.AccountNumber != nil {
		payload.AccountNumber = *bankAccountModel.AccountNumber
	}
	if bankAccountModel.IBAN != nil {
		payload.IBAN = *bankAccountModel.IBAN
	}
	if bankAccountModel.SwiftBicCode != nil {
		payload.SwiftBicCode = *bankAccountModel.SwiftBicCode
	}
	if bankAccountModel.Country != nil {
		payload.Country = *bankAccountModel.Country
	}

	for _, relatedAccount := range bankAccountModel.RelatedAccounts {
		payload.RelatedAccounts = append(payload.RelatedAccounts, internalEvents.BankAccountRelatedAccountsPayload{
			CreatedAt:   relatedAccount.CreatedAt,
			AccountID:   relatedAccount.AccountID.String(),
			ConnectorID: relatedAccount.AccountID.ConnectorID.String(),
			Provider:    models.ToV3Provider(relatedAccount.AccountID.ConnectorID.Provider),
		})
	}

	return payload
}

