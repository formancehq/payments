package storage

import (
	"context"
	"testing"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPaginateWithOffset(t *testing.T) {
	t.Parallel()

	store := newStore(t)
	ctx := context.Background()
	
	accounts := []models.Account{
		{
			ID: models.AccountID{
				Reference: "account1",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test",
				},
			},
			Name: "Test Account 1",
		},
		{
			ID: models.AccountID{
				Reference: "account2",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test",
				},
			},
			Name: "Test Account 2",
		},
		{
			ID: models.AccountID{
				Reference: "account3",
				ConnectorID: models.ConnectorID{
					Reference: uuid.New(),
					Provider:  "test",
				},
			},
			Name: "Test Account 3",
		},
	}
	
	err := store.AccountsUpsert(ctx, accounts)
	require.NoError(t, err)
	
	query := ListAccountsQuery{
		PageSize: 2,
		Cursor: bunpaginate.OffsetPaginatedQuery[models.ListAccountsFilter]{
			PageSize: 2,
			Order:    bunpaginate.OrderAsc,
		},
	}
	
	result, err := store.AccountsList(ctx, query)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, len(result.Data))
}
