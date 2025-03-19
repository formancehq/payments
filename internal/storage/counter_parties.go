package storage

import (
	"context"
	"fmt"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/go-libs/v2/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type counterParty struct {
	bun.BaseModel `bun:"table:counter_parties"`

	// Mandatory fields
	ID        uuid.UUID `bun:"id,pk,type:uuid,notnull"`
	CreatedAt time.Time `bun:"created_at,type:timestamp without time zone,notnull"`

	// Encrypted fields
	Name         string  `bun:"decrypted_name,scanonly"`
	StreetName   *string `bun:"decrypted_street_name,scanonly"`
	StreetNumber *string `bun:"decrypted_street_number,scanonly"`
	City         *string `bun:"decrypted_city,scanonly"`
	PostalCode   *string `bun:"decrypted_postal_code,scanonly"`
	Region       *string `bun:"decrypted_region,scanonly"`
	Country      *string `bun:"decrypted_country,scanonly"`
	Email        *string `bun:"decrypted_email,scanonly"`
	PhoneNumber  *string `bun:"decrypted_phone,scanonly"`

	// Optional fields
	BankAccountID *uuid.UUID `bun:"bank_account_id,type:uuid,nullzero"`

	// Optional fields with default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	RelatedAccounts []*counterPartiesRelatedAccount `bun:"rel:has-many,join:id=counter_party_id"`
}

func (s *store) CounterPartyUpsert(ctx context.Context, cp models.CounterParty, bankAccount *models.BankAccount) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return e("begin transaction: %w", err)
	}

	var errTx error
	defer func() {
		if errTx != nil {
			if err := tx.Rollback(); err != nil {
				s.logger.Errorf("failed to rollback transaction: %v", err)
			}
		}
	}()

	if bankAccount != nil {
		if err := s.insertBankAccountWithTx(ctx, tx, *bankAccount); err != nil {
			errTx = err
			return err
		}
	}

	toInsert := fromCounterPartyModels(cp)

	_, err = tx.NewRaw(`
		INSERT INTO counter_parties (id, created_at, bank_account_id, metadata, name, street_name, street_number, city, region, postal_code, country, email, phone)
		VALUES (?0, ?1, ?2, ?3,
			pgp_sym_encrypt(?4::TEXT, ?13, ?14),
			pgp_sym_encrypt(?5::TEXT, ?13, ?14),
			pgp_sym_encrypt(?6::TEXT, ?13, ?14),
			pgp_sym_encrypt(?7::TEXT, ?13, ?14),
			pgp_sym_encrypt(?8::TEXT, ?13, ?14),
			pgp_sym_encrypt(?9::TEXT, ?13, ?14),
			pgp_sym_encrypt(?10::TEXT, ?13, ?14),
			pgp_sym_encrypt(?11::TEXT, ?13, ?14),
			pgp_sym_encrypt(?12::TEXT, ?13, ?14)
		)
		ON CONFLICT (id) DO NOTHING
		RETURNING id
	`, toInsert.ID, toInsert.CreatedAt, toInsert.BankAccountID, toInsert.Metadata,
		toInsert.Name, toInsert.StreetName, toInsert.StreetNumber, toInsert.City,
		toInsert.Region, toInsert.PostalCode, toInsert.Country, toInsert.Email,
		toInsert.PhoneNumber, s.configEncryptionKey, encryptionOptions,
	).Exec(ctx)
	if err != nil {
		errTx = err
		return e("insert counter party: %w", err)
	}

	if len(toInsert.RelatedAccounts) > 0 {
		// Insert or update the related accounts
		_, err = tx.NewInsert().
			Model(&toInsert.RelatedAccounts).
			On("CONFLICT (counter_party_id, account_id) DO NOTHING").
			Exec(ctx)
		if err != nil {
			errTx = err
			return e("insert related accounts", err)
		}
	}

	if err := tx.Commit(); err != nil {
		errTx = err
		return e("commit transaction", tx.Commit())
	}

	return nil
}

func (s *store) CounterPartiesGet(ctx context.Context, id uuid.UUID) (*models.CounterParty, error) {
	var counterParty counterParty

	query := s.db.NewSelect().
		Model(&counterParty).
		Column("id", "created_at", "bank_account_id", "metadata").
		Where("id = ?", id).
		Relation("RelatedAccounts")
	query = s.counterPartiesSelectDecryptColumnExpr(query)

	err := query.
		Scan(ctx)
	if err != nil {
		return nil, e("select counter party: %w", err)
	}

	cp := toCounterPartyModels(counterParty)

	return &cp, nil
}

type CounterPartyQuery struct{}

type ListCounterPartiesQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[CounterPartyQuery]]

func NewListCounterPartiesQuery(opts bunpaginate.PaginatedQueryOptions[CounterPartyQuery]) ListCounterPartiesQuery {
	return ListCounterPartiesQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) counterPartiesQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "id", key == "bank_account_id":
			if operator != "$match" {
				return "", nil, fmt.Errorf("'%s' column can only be used with $match: %w", key, ErrValidation)
			}
			return fmt.Sprintf("%s = ?", key), []any{value}, nil
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
}

func (s *store) CounterPartiesList(ctx context.Context, q ListCounterPartiesQuery) (*bunpaginate.Cursor[models.CounterParty], error) {
	var (
		where string
		args  []any
		err   error
	)
	if q.Options.QueryBuilder != nil {
		where, args, err = s.counterPartiesQueryContext(q.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[CounterPartyQuery], counterParty](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[CounterPartyQuery]])(&q),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			query = query.Relation("RelatedAccounts")
			query = query.Column("id", "created_at", "bank_account_id", "metadata")
			query = s.counterPartiesSelectDecryptColumnExpr(query)

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

	counterParties := make([]models.CounterParty, 0, len(cursor.Data))
	for _, a := range cursor.Data {
		counterParties = append(counterParties, toCounterPartyModels(a))
	}

	return &bunpaginate.Cursor[models.CounterParty]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     counterParties,
	}, nil
}

func (s *store) counterPartiesSelectDecryptColumnExpr(query *bun.SelectQuery) *bun.SelectQuery {
	return query.
		ColumnExpr("pgp_sym_decrypt(name, ?, ?) as decrypted_name", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(street_name, ?, ?) as decrypted_street_name", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(street_number, ?, ?) as decrypted_street_number", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(city, ?, ?) as decrypted_city", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(region, ?, ?) as decrypted_region", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(postal_code, ?, ?) as decrypted_postal_code", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(country, ?, ?) as decrypted_country", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(email, ?, ?) as decrypted_email", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(phone, ?, ?) as decrypted_phone", s.configEncryptionKey, encryptionOptions)
}

type counterPartiesRelatedAccount struct {
	bun.BaseModel `bun:"table:counter_parties_related_accounts"`

	// Mandatory fields
	CounterPartyID uuid.UUID          `bun:"counter_party_id,pk,type:uuid,notnull"`
	AccountID      models.AccountID   `bun:"account_id,pk,type:character varying,notnull"`
	ConnectorID    models.ConnectorID `bun:"connector_id,type:character varying,notnull"`
	CreatedAt      time.Time          `bun:"created_at,type:timestamp without time zone,notnull"`
}

func (s *store) CounterPartiesAddRelatedAccount(ctx context.Context, cpID uuid.UUID, relatedAccount models.CounterPartiesRelatedAccount) error {
	toInsert := fromCounterPartyRelatedAccountModels(relatedAccount, cpID)

	_, err := s.db.NewInsert().
		Model(&toInsert).
		On("CONFLICT (counter_party_id, account_id) DO NOTHING").
		Exec(ctx)
	if err != nil {
		return e("add counter party related account", err)
	}

	return nil
}

func (s *store) CounterPartiesDeleteRelatedAccountFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	_, err := s.db.NewDelete().
		Model((*counterPartiesRelatedAccount)(nil)).
		Where("connector_id = ?", connectorID).
		Exec(ctx)
	if err != nil {
		return e("delete counter parties related account", err)
	}

	return nil
}

func fromCounterPartyModels(from models.CounterParty) counterParty {
	counterParty := counterParty{
		ID:            from.ID,
		CreatedAt:     time.New(from.CreatedAt).UTC(),
		Name:          from.Name,
		BankAccountID: from.BankAccountID,
		Metadata:      from.Metadata,
	}

	if from.Address != nil {
		counterParty.StreetName = &from.Address.StreetName
		counterParty.StreetNumber = &from.Address.StreetNumber
		counterParty.City = &from.Address.City
		counterParty.Region = &from.Address.Region
		counterParty.PostalCode = &from.Address.PostalCode
		counterParty.Country = &from.Address.Country
	}

	if from.ContactDetails != nil {
		counterParty.Email = from.ContactDetails.Email
		counterParty.PhoneNumber = from.ContactDetails.Phone
	}

	relatedAccounts := make([]*counterPartiesRelatedAccount, 0, len(from.RelatedAccounts))
	for _, ra := range from.RelatedAccounts {
		relatedAccounts = append(relatedAccounts, pointer.For(fromCounterPartyRelatedAccountModels(ra, from.ID)))
	}
	counterParty.RelatedAccounts = relatedAccounts

	return counterParty
}

func toCounterPartyModels(from counterParty) models.CounterParty {
	to := models.CounterParty{
		ID:            from.ID,
		CreatedAt:     from.CreatedAt.Time,
		Name:          from.Name,
		BankAccountID: from.BankAccountID,
		Metadata:      from.Metadata,
	}

	to.Address = fillAddress(from)
	to.ContactDetails = fillContactDetails(from)

	relatedAccounts := make([]models.CounterPartiesRelatedAccount, 0, len(from.RelatedAccounts))
	for _, ra := range from.RelatedAccounts {
		relatedAccounts = append(relatedAccounts, toCounterPartiesRelatedAccountModels(*ra))
	}
	to.RelatedAccounts = relatedAccounts

	return to
}

func fillAddress(from counterParty) *models.Address {
	if from.StreetName == nil && from.StreetNumber == nil && from.City == nil && from.PostalCode == nil && from.Region == nil && from.Country == nil {
		return nil
	}

	streetName := ""
	if from.StreetName != nil {
		streetName = *from.StreetName
	}

	streetNumber := ""
	if from.StreetNumber != nil {
		streetNumber = *from.StreetNumber
	}

	city := ""
	if from.City != nil {
		city = *from.City
	}

	postalCode := ""
	if from.PostalCode != nil {
		postalCode = *from.PostalCode
	}

	region := ""
	if from.Region != nil {
		region = *from.Region
	}

	country := ""
	if from.Country != nil {
		country = *from.Country
	}

	return &models.Address{
		StreetName:   streetName,
		StreetNumber: streetNumber,
		City:         city,
		PostalCode:   postalCode,
		Region:       region,
		Country:      country,
	}
}

func fillContactDetails(from counterParty) *models.ContactDetails {
	if from.Email == nil && from.PhoneNumber == nil {
		return nil
	}

	return &models.ContactDetails{
		Email: from.Email,
		Phone: from.PhoneNumber,
	}
}

func fromCounterPartyRelatedAccountModels(from models.CounterPartiesRelatedAccount, cpID uuid.UUID) counterPartiesRelatedAccount {
	return counterPartiesRelatedAccount{
		CounterPartyID: cpID,
		AccountID:      from.AccountID,
		ConnectorID:    from.AccountID.ConnectorID,
		CreatedAt:      time.New(from.CreatedAt),
	}
}

func toCounterPartiesRelatedAccountModels(from counterPartiesRelatedAccount) models.CounterPartiesRelatedAccount {
	return models.CounterPartiesRelatedAccount{
		AccountID: from.AccountID,
		CreatedAt: from.CreatedAt.Time,
	}
}
