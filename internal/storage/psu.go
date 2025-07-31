package storage

import (
	"context"
	"fmt"
	"github.com/formancehq/go-libs/v3/platform/postgres"
	"github.com/pkg/errors"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type paymentServiceUser struct {
	bun.BaseModel `bun:"payment_service_users"`

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

	// Optional
	Locale *string `bun:"locale,nullzero"`

	// Optional fields with default
	Metadata map[string]string `bun:"metadata,type:jsonb,nullzero,notnull,default:'{}'"`

	// Relations
	BankAccounts []bankAccount `bun:"rel:has-many,join:id=psu_id"`
}

func (s *store) PaymentServiceUsersCreate(ctx context.Context, psu models.PaymentServiceUser) error {
	paymentServiceUser, relatedBankAccounts := fromPaymentServiceUserModels(psu)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "begin transaction: %w")
	}

	var errTx error
	defer func() {
		rollbackOnTxError(ctx, &tx, errTx)
	}()

	_, err = tx.NewRaw(`
		INSERT INTO payment_service_users (id, created_at, metadata, locale, name, street_name, street_number, city, region, postal_code, country, email, phone_number)
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
	`, paymentServiceUser.ID, paymentServiceUser.CreatedAt, paymentServiceUser.Metadata,
		paymentServiceUser.Locale, paymentServiceUser.Name, paymentServiceUser.StreetName, paymentServiceUser.StreetNumber, paymentServiceUser.City,
		paymentServiceUser.Region, paymentServiceUser.PostalCode, paymentServiceUser.Country, paymentServiceUser.Email,
		paymentServiceUser.PhoneNumber, s.configEncryptionKey, encryptionOptions,
	).Exec(ctx)
	if err != nil {
		errTx = err
		return errors.Wrap(postgres.ResolveError(err), "insert psu: %w")
	}

	if len(relatedBankAccounts) > 0 {
		// Update related bank accounts
		for _, bankAccountID := range relatedBankAccounts {
			res, err := tx.NewUpdate().
				Model((*bankAccount)(nil)).
				Set("psu_id = ?", paymentServiceUser.ID).
				Where("id = ?", bankAccountID).
				Exec(ctx)
			if err != nil {
				errTx = err
				return errors.Wrap(postgres.ResolveError(err), "update bank account to add psu id")
			}

			rowsAffected, err := res.RowsAffected()
			if err != nil {
				errTx = err
				return errors.Wrap(postgres.ResolveError(err), "update bank account to add psu id")
			}

			if rowsAffected == 0 {
				errTx = ErrNotFound
				return errors.Wrap(ErrNotFound, "bank account")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		errTx = err
		return errors.Wrap(postgres.ResolveError(err), "commit transaction")
	}

	return nil
}

func (s *store) PaymentServiceUsersGet(ctx context.Context, id uuid.UUID) (*models.PaymentServiceUser, error) {
	var psu paymentServiceUser
	query := s.db.NewSelect().
		Model(&psu).
		Column("id", "created_at", "metadata", "locale").
		Where("id = ?", id).
		Relation("BankAccounts")

	query = s.paymentServiceUsersSelectDecryptColumnExpr(query)

	err := query.
		Scan(ctx)
	if err != nil {
		return nil, errors.Wrap(postgres.ResolveError(err), "select psu: %w")
	}

	res := toPaymentServiceUserModels(psu)

	return &res, nil
}

// TODO(polo): add tests
func (s *store) PaymentServiceUsersDelete(ctx context.Context, paymentServiceUserID string) error {
	res, err := s.db.NewDelete().
		Model((*paymentServiceUser)(nil)).
		Where("id = ?", paymentServiceUserID).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "delete psu")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "delete psu")
	}

	if rowsAffected == 0 {
		return errors.Wrap(ErrNotFound, "psu")
	}

	return nil
}

type PSUQuery struct{}

type ListPSUsQuery bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUQuery]]

func NewListPSUQuery(opts bunpaginate.PaginatedQueryOptions[PSUQuery]) ListPSUsQuery {
	return ListPSUsQuery{
		Order:    bunpaginate.OrderAsc,
		PageSize: opts.PageSize,
		Options:  opts,
	}
}

func (s *store) paymentServiceUsersQueryContext(qb query.Builder) (string, []any, error) {
	return qb.Build(query.ContextFn(func(key, operator string, value any) (string, []any, error) {
		switch {
		case key == "id":
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

func (s *store) PaymentServiceUsersList(ctx context.Context, query ListPSUsQuery) (*bunpaginate.Cursor[models.PaymentServiceUser], error) {
	var (
		where string
		args  []any
		err   error
	)
	if query.Options.QueryBuilder != nil {
		where, args, err = s.paymentServiceUsersQueryContext(query.Options.QueryBuilder)
		if err != nil {
			return nil, err
		}
	}

	cursor, err := paginateWithOffset[bunpaginate.PaginatedQueryOptions[PSUQuery], paymentServiceUser](s, ctx,
		(*bunpaginate.OffsetPaginatedQuery[bunpaginate.PaginatedQueryOptions[PSUQuery]])(&query),
		func(query *bun.SelectQuery) *bun.SelectQuery {
			query = query.Relation("BankAccounts")
			query = query.Column("id", "created_at", "metadata", "locale")
			query = s.paymentServiceUsersSelectDecryptColumnExpr(query)

			if where != "" {
				query = query.Where(where, args...)
			}

			query = query.Order("created_at DESC", "sort_id DESC")

			return query
		},
	)
	if err != nil {
		return nil, errors.Wrap(postgres.ResolveError(err), "failed to fetch accounts")
	}

	counterParties := make([]models.PaymentServiceUser, 0, len(cursor.Data))
	for _, a := range cursor.Data {
		counterParties = append(counterParties, toPaymentServiceUserModels(a))
	}

	return &bunpaginate.Cursor[models.PaymentServiceUser]{
		PageSize: cursor.PageSize,
		HasMore:  cursor.HasMore,
		Previous: cursor.Previous,
		Next:     cursor.Next,
		Data:     counterParties,
	}, nil
}

func (s *store) PaymentServiceUsersAddBankAccount(ctx context.Context, psuID, bankAccountID uuid.UUID) error {
	res, err := s.db.NewUpdate().
		Model((*bankAccount)(nil)).
		Set("psu_id = ?", psuID).
		Where("id = ?", bankAccountID).
		Exec(ctx)
	if err != nil {
		pgErr := postgres.ResolveError(err)
		if (postgres.ErrFKConstraintFailed{}.Is(pgErr)) {
			return ErrForeignKeyViolation
		}
		return errors.Wrap(postgres.ResolveError(err), "update bank account to add psu id")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(postgres.ResolveError(err), "update bank account to add psu id")
	}

	if rowsAffected == 0 {
		return errors.Wrap(ErrNotFound, "bank account")
	}

	return nil
}

func (s *store) paymentServiceUsersSelectDecryptColumnExpr(query *bun.SelectQuery) *bun.SelectQuery {
	return query.
		ColumnExpr("pgp_sym_decrypt(name, ?, ?) as decrypted_name", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(street_name, ?, ?) as decrypted_street_name", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(street_number, ?, ?) as decrypted_street_number", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(city, ?, ?) as decrypted_city", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(region, ?, ?) as decrypted_region", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(postal_code, ?, ?) as decrypted_postal_code", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(country, ?, ?) as decrypted_country", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(email, ?, ?) as decrypted_email", s.configEncryptionKey, encryptionOptions).
		ColumnExpr("pgp_sym_decrypt(phone_number, ?, ?) as decrypted_phone", s.configEncryptionKey, encryptionOptions)
}

func fromPaymentServiceUserModels(from models.PaymentServiceUser) (paymentServiceUser, []uuid.UUID) {
	psu := paymentServiceUser{
		ID:        from.ID,
		CreatedAt: time.New(from.CreatedAt),
		Name:      from.Name,
		Metadata:  from.Metadata,
	}

	if from.Address != nil {
		psu.StreetName = from.Address.StreetName
		psu.StreetNumber = from.Address.StreetNumber
		psu.City = from.Address.City
		psu.PostalCode = from.Address.PostalCode
		psu.Region = from.Address.Region
		psu.Country = from.Address.Country
	}

	if from.ContactDetails != nil {
		psu.Email = from.ContactDetails.Email
		psu.PhoneNumber = from.ContactDetails.PhoneNumber
		psu.Locale = from.ContactDetails.Locale
	}

	return psu, from.BankAccountIDs
}

func toPaymentServiceUserModels(from paymentServiceUser) models.PaymentServiceUser {
	psu := models.PaymentServiceUser{
		ID:        from.ID,
		CreatedAt: from.CreatedAt.Time,
		Name:      from.Name,
		Metadata:  from.Metadata,
	}

	psu.Address = fillAddress(from)
	psu.ContactDetails = fillContactDetails(from)

	psu.BankAccountIDs = make([]uuid.UUID, len(from.BankAccounts))
	for i, bankAccount := range from.BankAccounts {
		psu.BankAccountIDs[i] = bankAccount.ID
	}

	return psu
}

func fillAddress(from paymentServiceUser) *models.Address {
	if from.StreetName == nil && from.StreetNumber == nil && from.City == nil && from.PostalCode == nil && from.Region == nil && from.Country == nil {
		return nil
	}

	return &models.Address{
		StreetName:   from.StreetName,
		StreetNumber: from.StreetNumber,
		City:         from.City,
		PostalCode:   from.PostalCode,
		Region:       from.Region,
		Country:      from.Country,
	}
}

func fillContactDetails(from paymentServiceUser) *models.ContactDetails {
	if from.Email == nil && from.PhoneNumber == nil {
		return nil
	}

	return &models.ContactDetails{
		Email:       from.Email,
		PhoneNumber: from.PhoneNumber,
		Locale:      from.Locale,
	}
}
