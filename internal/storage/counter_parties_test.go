package storage

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/go-libs/v2/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	defaultCounterParty = models.CounterParty{
		ID:        uuid.New(),
		Name:      "test",
		CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
		ContactDetails: &models.ContactDetails{
			Email: pointer.For("test"),
			Phone: pointer.For("test"),
		},
		Address: &models.Address{
			StreetName:   pointer.For("test"),
			StreetNumber: pointer.For("test"),
			City:         pointer.For("test"),
			PostalCode:   pointer.For("test"),
			Country:      pointer.For("test"),
		},
		BankAccountID: &defaultBankAccount.ID,
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	defaultCounterParty2 = models.CounterParty{
		ID:        uuid.New(),
		Name:      "test2",
		CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
		RelatedAccounts: []models.CounterPartiesRelatedAccount{
			{
				AccountID: defaultAccounts()[0].ID,
				CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
			},
		},
	}

	defaultCounterParty3 = models.CounterParty{
		ID:        uuid.New(),
		Name:      "test",
		CreatedAt: now.Add(-55 * time.Minute).UTC().Time,
		ContactDetails: &models.ContactDetails{
			Email: pointer.For("test"),
		},
		Address: &models.Address{
			StreetName: pointer.For("test"),
			PostalCode: pointer.For("test"),
			Country:    pointer.For("test"),
		},
		BankAccountID: &defaultBankAccount.ID,
	}
)

func upsertCounterParty(t *testing.T, ctx context.Context, storage Storage, counterParty models.CounterParty, ba *models.BankAccount) {
	require.NoError(t, storage.CounterPartyUpsert(ctx, counterParty, ba))
}

func TestCounterPartiesUpsert(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertCounterParty(t, ctx, store, defaultCounterParty, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty2, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty3, nil)

	t.Run("upsert with same id", func(t *testing.T) {
		cp := models.CounterParty{
			ID:        defaultCounterParty.ID,
			Name:      "changed",
			CreatedAt: now.Add(-40 * time.Minute).UTC().Time,
			ContactDetails: &models.ContactDetails{
				Email: pointer.For("changed"),
				Phone: pointer.For("changed"),
			},
			Address: &models.Address{
				StreetName:   pointer.For("changed"),
				StreetNumber: pointer.For("changed"),
				City:         pointer.For("changed"),
				PostalCode:   pointer.For("changed"),
				Country:      pointer.For("changed"),
			},
		}

		require.NoError(t, store.CounterPartyUpsert(ctx, cp, nil))

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty.ID)
		require.NoError(t, err)
		// Should not update the counter party
		compareCounterParties(t, defaultCounterParty, *actual)
	})

	t.Run("unknown bank account id", func(t *testing.T) {
		cp := models.CounterParty{
			ID:             uuid.New(),
			Name:           "test",
			CreatedAt:      now.Add(-60 * time.Minute).UTC().Time,
			ContactDetails: &models.ContactDetails{},
			Address:        &models.Address{},
			BankAccountID:  pointer.For(uuid.New()),
		}

		require.Error(t, store.CounterPartyUpsert(ctx, cp, nil))
	})

	t.Run("upsert with new bank account", func(t *testing.T) {
		ba := models.BankAccount{
			ID:            uuid.New(),
			CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
			Name:          "test-bank-accounts-counter-party",
			AccountNumber: pointer.For("12345678"),
			Country:       pointer.For("US"),
			Metadata: map[string]string{
				"foo": "bar",
			},
		}

		cp := models.CounterParty{
			ID:            uuid.New(),
			Name:          "test",
			CreatedAt:     now.Add(-60 * time.Minute).UTC().Time,
			BankAccountID: &ba.ID,
		}

		require.NoError(t, store.CounterPartyUpsert(ctx, cp, &ba))

		actual, err := store.CounterPartiesGet(ctx, cp.ID)
		require.NoError(t, err)
		compareCounterParties(t, cp, *actual)
	})
}

func TestCounterPartiesGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertCounterParty(t, ctx, store, defaultCounterParty, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty2, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty3, nil)

	t.Run("get counter party will all fields filled", func(t *testing.T) {
		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty.ID)
		require.NoError(t, err)
		compareCounterParties(t, defaultCounterParty, *actual)
	})

	t.Run("get counter party with only required fields", func(t *testing.T) {
		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty2.ID)
		require.NoError(t, err)
		compareCounterParties(t, defaultCounterParty2, *actual)
	})

	t.Run("get counter party with only required fields and some optional fields", func(t *testing.T) {
		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty3.ID)
		require.NoError(t, err)
		compareCounterParties(t, defaultCounterParty3, *actual)
	})
}

func TestCounterPartiesList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertCounterParty(t, ctx, store, defaultCounterParty, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty2, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty3, nil)

	t.Run("wrong query builder when listing by id", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("id", "test1")),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("wrong query builder when listing by bank account id", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("bank_account_id", "test1")),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list counter parties by id", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultCounterParty.ID.String())),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty, cursor.Data[0])
	})

	t.Run("list counter parties by unknown id", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", uuid.New())),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list counter parties by bank account id", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("bank_account_id", defaultBankAccount.ID.String())),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 2)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty3, cursor.Data[0])
		compareCounterParties(t, defaultCounterParty, cursor.Data[1])
	})

	t.Run("list counter parties by unknown bank account id", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("bank_account_id", uuid.New())),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("lsit counter parties by metadata", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty, cursor.Data[0])
	})

	t.Run("lsit counter parties by unknown metadata", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[unknown]", "bar")),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("lsit counter parties by metadata", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty, cursor.Data[0])
	})

	t.Run("list counter parties test cursor", func(t *testing.T) {
		q := NewListCounterPartiesQuery(
			bunpaginate.NewPaginatedQueryOptions(CounterPartyQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty2, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.CounterPartiesList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		compareCounterParties(t, defaultCounterParty2, cursor.Data[0])
	})
}

func TestCounterPartiesAddRelatedAccount(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertCounterParty(t, ctx, store, defaultCounterParty, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty2, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty3, nil)

	t.Run("add related account when empty", func(t *testing.T) {
		acc := models.CounterPartiesRelatedAccount{
			AccountID: defaultAccounts()[0].ID,
			CreatedAt: now.UTC().Time,
		}

		cp := defaultCounterParty
		cp.RelatedAccounts = append(cp.RelatedAccounts, acc)

		require.NoError(t, store.CounterPartiesAddRelatedAccount(ctx, cp.ID, acc))

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty.ID)
		require.NoError(t, err)
		compareCounterParties(t, cp, *actual)
	})

	t.Run("add related account when not empty", func(t *testing.T) {
		acc := models.CounterPartiesRelatedAccount{
			AccountID: defaultAccounts()[1].ID,
			CreatedAt: now.UTC().Time,
		}

		cp := defaultCounterParty2
		cp.RelatedAccounts = append(cp.RelatedAccounts, acc)

		require.NoError(t, store.CounterPartiesAddRelatedAccount(ctx, defaultCounterParty2.ID, acc))

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty2.ID)
		require.NoError(t, err)
		compareCounterParties(t, cp, *actual)
	})

	t.Run("add related account with unknown bank account", func(t *testing.T) {
		acc := models.CounterPartiesRelatedAccount{
			AccountID: defaultAccounts()[1].ID,
			CreatedAt: now.UTC().Time,
		}

		require.Error(t, store.CounterPartiesAddRelatedAccount(ctx, uuid.New(), acc))
	})

	t.Run("add related account with unknown connector", func(t *testing.T) {
		acc := models.CounterPartiesRelatedAccount{
			AccountID: models.AccountID{
				Reference: "unknown",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "unknown",
				},
			},
			CreatedAt: now.UTC().Time,
		}

		require.Error(t, store.CounterPartiesAddRelatedAccount(ctx, defaultCounterParty.ID, acc))
	})

	t.Run("add related account with existing related account", func(t *testing.T) {
		acc := models.CounterPartiesRelatedAccount{
			AccountID: defaultAccounts()[0].ID,
			CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
		}

		cp := defaultCounterParty3
		cp.RelatedAccounts = append(cp.RelatedAccounts, acc)

		require.NoError(t, store.CounterPartiesAddRelatedAccount(ctx, defaultCounterParty3.ID, acc))

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty3.ID)
		require.NoError(t, err)
		compareCounterParties(t, cp, *actual)

		require.NoError(t, store.CounterPartiesAddRelatedAccount(ctx, defaultCounterParty3.ID, acc))

		actual, err = store.CounterPartiesGet(ctx, defaultCounterParty3.ID)
		require.NoError(t, err)
		compareCounterParties(t, cp, *actual)
	})
}

func TestCounterPartiesDeleteRelatedAccountFromConnectorID(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertAccounts(t, ctx, store, defaultAccounts())
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertCounterParty(t, ctx, store, defaultCounterParty, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty2, nil)
	upsertCounterParty(t, ctx, store, defaultCounterParty3, nil)

	t.Run("delete related account with unknown connector", func(t *testing.T) {
		require.NoError(t, store.CounterPartiesDeleteRelatedAccountFromConnectorID(ctx, models.ConnectorID{
			Reference: uuid.New(),
			Provider:  "unknown",
		}))

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty2.ID)
		require.NoError(t, err)
		compareCounterParties(t, defaultCounterParty2, *actual)
	})

	t.Run("delete related account with another connector id", func(t *testing.T) {
		require.NoError(t, store.CounterPartiesDeleteRelatedAccountFromConnectorID(ctx, defaultConnector2.ID))

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty2.ID)
		require.NoError(t, err)
		compareCounterParties(t, defaultCounterParty2, *actual)
	})

	t.Run("delete related account", func(t *testing.T) {
		require.NoError(t, store.CounterPartiesDeleteRelatedAccountFromConnectorID(ctx, defaultConnector.ID))

		cp := defaultCounterParty2
		cp.RelatedAccounts = nil

		actual, err := store.CounterPartiesGet(ctx, defaultCounterParty2.ID)
		require.NoError(t, err)
		compareCounterParties(t, cp, *actual)
	})
}

func compareCounterParties(t *testing.T, expected, actual models.CounterParty) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t, expected.Name, actual.Name)

	require.Equal(t, len(expected.Metadata), len(actual.Metadata))
	for k, v := range expected.Metadata {
		require.Equal(t, v, actual.Metadata[k])
	}
	for k, v := range actual.Metadata {
		require.Equal(t, v, expected.Metadata[k])
	}

	switch {
	case expected.BankAccountID != nil && actual.BankAccountID != nil:
		require.Equal(t, *expected.BankAccountID, *actual.BankAccountID)
	case expected.BankAccountID == nil && actual.BankAccountID == nil:
		// Nothing to do
	default:
		require.Fail(t, "BankAccountID is different")
	}

	compareCounterPartiesAddressed(t, expected.Address, actual.Address)
	compareCounterPartiesContactDetails(t, expected.ContactDetails, actual.ContactDetails)

	require.Equal(t, len(expected.RelatedAccounts), len(actual.RelatedAccounts))
	for i := range expected.RelatedAccounts {
		require.Equal(t, expected.RelatedAccounts[i], actual.RelatedAccounts[i])
	}
}

func compareCounterPartiesAddressed(t *testing.T, expected, actual *models.Address) {
	switch {
	case expected == nil && actual == nil:
		return
	case expected != nil && actual != nil:
		// Do the next tests
	default:
		require.Fail(t, "Address is different")
	}

	compareInterface(t, "StreetName", expected.StreetName, actual.StreetName)
	compareInterface(t, "StreetNumber", expected.StreetNumber, actual.StreetNumber)
	compareInterface(t, "City", expected.City, actual.City)
	compareInterface(t, "PostalCode", expected.PostalCode, actual.PostalCode)
	compareInterface(t, "Country", expected.Country, actual.Country)
}

func compareCounterPartiesContactDetails(t *testing.T, expected, actual *models.ContactDetails) {
	switch {
	case expected == nil && actual == nil:
		return
	case expected != nil && actual != nil:
		// Do the next tests
	default:
		require.Fail(t, "ContactDetails is different")
	}

	compareInterface(t, "Email", expected.Email, actual.Email)
	compareInterface(t, "Phone", expected.Phone, actual.Phone)
}

func compareInterface(t *testing.T, name string, expected, actual interface{}) {
	switch {
	case expected == nil && actual == nil:
		return
	case expected != nil && actual != nil:
		// Do the next tests
	default:
		require.Failf(t, "%s field is different", name)
	}

	require.Equal(t, expected, actual)
}
