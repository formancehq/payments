package storage

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/query"
	"github.com/formancehq/go-libs/v3/time"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	defaultPSU = models.PaymentServiceUser{
		ID:        uuid.New(),
		Name:      "test",
		CreatedAt: now.Add(-60 * time.Minute).UTC().Time,
		ContactDetails: &models.ContactDetails{
			Email:       pointer.For("test"),
			PhoneNumber: pointer.For("test"),
		},
		Address: &models.Address{
			StreetName:   pointer.For("test"),
			StreetNumber: pointer.For("test"),
			City:         pointer.For("test"),
			Region:       pointer.For("test"),
			PostalCode:   pointer.For("test"),
			Country:      pointer.For("test"),
		},
		BankAccountIDs: []uuid.UUID{defaultBankAccount.ID},
		Metadata: map[string]string{
			"foo": "bar",
		},
	}

	defaultPSU2 = models.PaymentServiceUser{
		ID:        uuid.New(),
		Name:      "test2",
		CreatedAt: now.Add(-30 * time.Minute).UTC().Time,
	}

	defaultPSU3 = models.PaymentServiceUser{
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
		BankAccountIDs: []uuid.UUID{defaultBankAccount2.ID},
	}
)

func createPSU(t *testing.T, ctx context.Context, storage Storage, psu models.PaymentServiceUser) {
	require.NoError(t, storage.PaymentServiceUsersCreate(ctx, psu))
}

func TestPSUCreate(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)

	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertAccounts(t, ctx, store, defaultAccounts())
	createPSU(t, ctx, store, defaultPSU)

	t.Run("upsert with same id", func(t *testing.T) {
		psu := models.PaymentServiceUser{
			ID:        defaultPSU.ID,
			Name:      "changed",
			CreatedAt: now.Add(-40 * time.Minute).UTC().Time,
			ContactDetails: &models.ContactDetails{
				Email:       pointer.For("changed"),
				PhoneNumber: pointer.For("changed"),
			},
			Address: &models.Address{
				StreetName:   pointer.For("changed"),
				StreetNumber: pointer.For("changed"),
				City:         pointer.For("changed"),
				Region:       pointer.For("changed"),
				PostalCode:   pointer.For("changed"),
				Country:      pointer.For("changed"),
			},
		}

		require.NoError(t, store.PaymentServiceUsersCreate(ctx, psu))

		actual, err := store.PaymentServiceUsersGet(ctx, defaultPSU.ID)
		require.NoError(t, err)
		// Should not update the counter party
		comparePSUs(t, defaultPSU, *actual)
	})

	t.Run("unknown bank account id id", func(t *testing.T) {
		cp := models.PaymentServiceUser{
			ID:             uuid.New(),
			Name:           "test",
			CreatedAt:      now.Add(-60 * time.Minute).UTC().Time,
			ContactDetails: &models.ContactDetails{},
			Address:        &models.Address{},
			BankAccountIDs: []uuid.UUID{uuid.New()},
		}

		require.Error(t, store.PaymentServiceUsersCreate(ctx, cp))
	})
}

func TestPSUGet(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertAccounts(t, ctx, store, defaultAccounts())
	createPSU(t, ctx, store, defaultPSU)
	createPSU(t, ctx, store, defaultPSU2)
	createPSU(t, ctx, store, defaultPSU3)

	t.Run("get psu will all fields filled", func(t *testing.T) {
		actual, err := store.PaymentServiceUsersGet(ctx, defaultPSU.ID)
		require.NoError(t, err)
		comparePSUs(t, defaultPSU, *actual)
	})

	t.Run("get psu with only required fields", func(t *testing.T) {
		actual, err := store.PaymentServiceUsersGet(ctx, defaultPSU2.ID)
		require.NoError(t, err)
		comparePSUs(t, defaultPSU2, *actual)
	})

	t.Run("get psu with only required fields and some optional fields", func(t *testing.T) {
		actual, err := store.PaymentServiceUsersGet(ctx, defaultPSU3.ID)
		require.NoError(t, err)
		comparePSUs(t, defaultPSU3, *actual)
	})
}

func TestPSUList(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertAccounts(t, ctx, store, defaultAccounts())
	createPSU(t, ctx, store, defaultPSU)
	createPSU(t, ctx, store, defaultPSU2)
	createPSU(t, ctx, store, defaultPSU3)

	t.Run("wrong query builder when listing by id", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Lt("id", "test1")),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.Error(t, err)
		require.Nil(t, cursor)
	})

	t.Run("list psu by id", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", defaultPSU.ID.String())),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePSUs(t, defaultPSU, cursor.Data[0])
	})

	t.Run("list psu by unknown id", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("id", uuid.New())),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Empty(t, cursor.Data)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list psu by metadata", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePSUs(t, defaultPSU, cursor.Data[0])
	})

	t.Run("list psu by unknown metadata", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[unknown]", "bar")),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 0)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
	})

	t.Run("list psu by metadata", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(15).
				WithQueryBuilder(query.Match("metadata[foo]", "bar")),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePSUs(t, defaultPSU, cursor.Data[0])
	})

	t.Run("list psu test cursor", func(t *testing.T) {
		q := NewListPSUQuery(
			bunpaginate.NewPaginatedQueryOptions(PSUQuery{}).
				WithPageSize(1),
		)

		cursor, err := store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePSUs(t, defaultPSU2, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePSUs(t, defaultPSU3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Next, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.False(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.Empty(t, cursor.Next)
		comparePSUs(t, defaultPSU, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.NotEmpty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePSUs(t, defaultPSU3, cursor.Data[0])

		err = bunpaginate.UnmarshalCursor(cursor.Previous, &q)
		require.NoError(t, err)
		cursor, err = store.PaymentServiceUsersList(ctx, q)
		require.NoError(t, err)
		require.Len(t, cursor.Data, 1)
		require.True(t, cursor.HasMore)
		require.Empty(t, cursor.Previous)
		require.NotEmpty(t, cursor.Next)
		comparePSUs(t, defaultPSU2, cursor.Data[0])
	})
}

func TestPSUAddBankAccount(t *testing.T) {
	t.Parallel()

	ctx := logging.TestingContext()
	store := newStore(t)
	upsertConnector(t, ctx, store, defaultConnector)
	upsertBankAccount(t, ctx, store, defaultBankAccount)
	upsertBankAccount(t, ctx, store, defaultBankAccount2)
	upsertAccounts(t, ctx, store, defaultAccounts())
	createPSU(t, ctx, store, defaultPSU)

	t.Run("add bank account to psu", func(t *testing.T) {
		err := store.PaymentServiceUsersAddBankAccount(ctx, defaultPSU.ID, defaultBankAccount2.ID)
		require.NoError(t, err)

		actual, err := store.PaymentServiceUsersGet(ctx, defaultPSU.ID)
		require.NoError(t, err)
		require.Len(t, actual.BankAccountIDs, 2)
		require.Equal(t, defaultBankAccount.ID, actual.BankAccountIDs[0])
		require.Equal(t, defaultBankAccount2.ID, actual.BankAccountIDs[1])
	})

	t.Run("add unknown account to psu", func(t *testing.T) {
		err := store.PaymentServiceUsersAddBankAccount(ctx, defaultPSU.ID, uuid.New())
		require.Error(t, err)

		actual, err := store.PaymentServiceUsersGet(ctx, defaultPSU.ID)
		require.NoError(t, err)
		require.Len(t, actual.BankAccountIDs, 2)
		require.Equal(t, defaultBankAccount.ID, actual.BankAccountIDs[0])
		require.Equal(t, defaultBankAccount2.ID, actual.BankAccountIDs[1])
	})
}

func comparePSUs(t *testing.T, expected, actual models.PaymentServiceUser) {
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

	require.Equal(t, len(expected.BankAccountIDs), len(actual.BankAccountIDs))
	for i := range expected.BankAccountIDs {
		require.Equal(t, expected.BankAccountIDs[i], actual.BankAccountIDs[i])
	}

	compareCounterPartiesAddressed(t, expected.Address, actual.Address)
	compareCounterPartiesContactDetails(t, expected.ContactDetails, actual.ContactDetails)
}

func compareCounterPartiesAddressed(t *testing.T, expected, actual *models.Address) {
	switch {
	case expected == nil && actual == nil:
		return
	case expected != nil && actual != nil:
		compareInterface(t, "StreetName", expected.StreetName, actual.StreetName)
		compareInterface(t, "StreetNumber", expected.StreetNumber, actual.StreetNumber)
		compareInterface(t, "City", expected.City, actual.City)
		compareInterface(t, "Region", expected.Region, actual.Region)
		compareInterface(t, "PostalCode", expected.PostalCode, actual.PostalCode)
		compareInterface(t, "Country", expected.Country, actual.Country)
	default:
		require.Fail(t, "Address is different")
	}
}

func compareCounterPartiesContactDetails(t *testing.T, expected, actual *models.ContactDetails) {
	switch {
	case expected == nil && actual == nil:
		return
	case expected != nil && actual != nil:
		compareInterface(t, "Email", expected.Email, actual.Email)
		compareInterface(t, "Phone", expected.PhoneNumber, actual.PhoneNumber)
	default:
		require.Fail(t, "ContactDetails is different")
	}

}

func compareInterface(t *testing.T, name string, expected, actual *string) {
	switch {
	case expected == nil && actual == nil:
		return
	case expected != nil && actual != nil:
		require.Equal(t, expected, actual)
	default:
		require.Failf(t, "%s field is different", name)
	}

}
