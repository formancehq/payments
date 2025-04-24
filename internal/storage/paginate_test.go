package storage

import (
	"context"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type testModel struct {
	bun.BaseModel `bun:"table:test_models"`

	ID        string    `bun:"id,pk"`
	Name      string    `bun:"name"`
	CreatedAt time.Time `bun:"created_at"`
}

type testFilter struct {
	Name string `schema:"name"`
}

func TestPaginateWithOffset(t *testing.T) {
	t.Parallel()

	s := newStore(t)
	ctx := context.Background()
	
	db := s.(interface{ GetDB() *bun.DB }).GetDB()
	_, err := db.NewCreateTable().Model((*testModel)(nil)).Exec(ctx)
	require.NoError(t, err)
	
	testModels := []testModel{
		{ID: "1", Name: "Test 1", CreatedAt: time.Now()},
		{ID: "2", Name: "Test 2", CreatedAt: time.Now()},
		{ID: "3", Name: "Test 3", CreatedAt: time.Now()},
	}
	
	_, err = db.NewInsert().Model(&testModels).Exec(ctx)
	require.NoError(t, err)
	
	query := bunpaginate.OffsetPaginatedQuery[testFilter]{
		PageSize: 2,
		Order:    bunpaginate.OrderAsc,
	}
	
	storeImpl := s.(*store)
	result, err := paginateWithOffset[testFilter, testModel](storeImpl, ctx, &query, func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Model((*testModel)(nil)).Order("id ASC")
	})
	
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, len(result.Data))
	require.Equal(t, "1", result.Data[0].ID)
	require.Equal(t, "2", result.Data[1].ID)
}
