package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestConnectorID() models.ConnectorID {
	return models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}
}

func TestScopedAccountLookup_ListAccountsByConnector(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("empty result", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		store := storage.NewMockStorage(ctrl)
		connectorID := newTestConnectorID()

		store.EXPECT().
			AccountsListAllByConnectorID(ctx, connectorID).
			Return([]models.Account{}, nil)

		lookup := newScopedAccountLookup(store, connectorID)
		got, err := lookup.ListAccountsByConnector(ctx)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("happy path converts accounts to PSPAccount", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		store := storage.NewMockStorage(ctrl)
		connectorID := newTestConnectorID()

		accounts := []models.Account{
			{
				ID:           models.AccountID{Reference: "acc-1", ConnectorID: connectorID},
				ConnectorID:  connectorID,
				Reference:    "acc-1",
				DefaultAsset: pointer.For("BTC/8"),
				Raw:          []byte(`{}`),
			},
			{
				ID:           models.AccountID{Reference: "acc-2", ConnectorID: connectorID},
				ConnectorID:  connectorID,
				Reference:    "acc-2",
				DefaultAsset: pointer.For("USD/2"),
				Raw:          []byte(`{}`),
			},
		}

		store.EXPECT().
			AccountsListAllByConnectorID(ctx, connectorID).
			Return(accounts, nil)

		lookup := newScopedAccountLookup(store, connectorID)
		got, err := lookup.ListAccountsByConnector(ctx)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "acc-1", got[0].Reference)
		assert.Equal(t, "BTC/8", *got[0].DefaultAsset)
		assert.Equal(t, "acc-2", got[1].Reference)
		assert.Equal(t, "USD/2", *got[1].DefaultAsset)
	})

	t.Run("storage error propagates", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		store := storage.NewMockStorage(ctrl)
		connectorID := newTestConnectorID()

		boom := errors.New("boom")
		store.EXPECT().
			AccountsListAllByConnectorID(ctx, connectorID).
			Return(nil, boom)

		lookup := newScopedAccountLookup(store, connectorID)
		got, err := lookup.ListAccountsByConnector(ctx)

		require.ErrorIs(t, err, boom)
		assert.Nil(t, got)
	})
}

func TestNewAccountLookupFactory(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := storage.NewMockStorage(ctrl)

	factory := NewAccountLookupFactory(store)
	require.NotNil(t, factory)

	connectorA := newTestConnectorID()
	connectorB := newTestConnectorID()

	store.EXPECT().
		AccountsListAllByConnectorID(gomock.Any(), connectorA).
		Return([]models.Account{}, nil)
	store.EXPECT().
		AccountsListAllByConnectorID(gomock.Any(), connectorB).
		Return([]models.Account{}, nil)

	lookupA := factory(connectorA)
	lookupB := factory(connectorB)

	_, err := lookupA.ListAccountsByConnector(context.Background())
	require.NoError(t, err)
	_, err = lookupB.ListAccountsByConnector(context.Background())
	require.NoError(t, err)
}
