package services

import (
	"context"
	"errors"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/storage"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// errPaginationStuck stops the mocked storage from serving the same page
// forever when a caller never follows cursor.Next.
var errPaginationStuck = errors.New("pagination did not advance: the same page was requested again")

// pagedOffsets serves total items in pages of pageSize, keyed on the offset of
// the query the caller sends, the way paginateWithOffset does.
func pagedOffsets(t *testing.T, offset, pageSize uint64, total int, seen map[uint64]int) (from, to int, hasNext bool, err error) {
	t.Helper()
	seen[offset]++
	if seen[offset] > 1 {
		return 0, 0, false, errPaginationStuck
	}
	from = int(offset)
	to = from + int(pageSize)
	if to >= total {
		return from, total, false, nil
	}
	return from, to, true, nil
}

func TestPaymentInitiationAdjustmentsListAllFollowsCursor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := storage.NewMockStorage(ctrl)
	s := New(store, engine.NewMockEngine(ctrl), false)

	pid := models.PaymentInitiationID{Reference: "ref"}
	const total = 120
	seen := map[uint64]int{}

	store.EXPECT().PaymentInitiationAdjustmentsList(gomock.Any(), pid, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ models.PaymentInitiationID, q storage.ListPaymentInitiationAdjustmentsQuery) (*paginate.Cursor[models.PaymentInitiationAdjustment], error) {
			from, to, hasNext, err := pagedOffsets(t, q.Offset, q.PageSize, total, seen)
			if err != nil {
				return nil, err
			}
			data := make([]models.PaymentInitiationAdjustment, to-from)
			cursor := &paginate.Cursor[models.PaymentInitiationAdjustment]{PageSize: int(q.PageSize), HasMore: hasNext, Data: data}
			if hasNext {
				next := q
				next.Offset = uint64(to)
				cursor.Next = paginate.EncodeCursor(next)
			}
			return cursor, nil
		}).AnyTimes()

	adjustments, err := s.PaymentInitiationAdjustmentsListAll(context.Background(), pid)
	require.NoError(t, err)
	require.Len(t, adjustments, total)
}

func TestPaymentInitiationRelatedPaymentsListAllFollowsCursor(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := storage.NewMockStorage(ctrl)
	s := New(store, engine.NewMockEngine(ctrl), false)

	pid := models.PaymentInitiationID{Reference: "ref"}
	const total = 75
	seen := map[uint64]int{}

	store.EXPECT().PaymentInitiationRelatedPaymentsList(gomock.Any(), pid, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ models.PaymentInitiationID, q storage.ListPaymentInitiationRelatedPaymentsQuery) (*paginate.Cursor[models.Payment], error) {
			from, to, hasNext, err := pagedOffsets(t, q.Offset, q.PageSize, total, seen)
			if err != nil {
				return nil, err
			}
			data := make([]models.Payment, to-from)
			cursor := &paginate.Cursor[models.Payment]{PageSize: int(q.PageSize), HasMore: hasNext, Data: data}
			if hasNext {
				next := q
				next.Offset = uint64(to)
				cursor.Next = paginate.EncodeCursor(next)
			}
			return cursor, nil
		}).AnyTimes()

	payments, err := s.PaymentInitiationRelatedPaymentsListAll(context.Background(), pid)
	require.NoError(t, err)
	require.Len(t, payments, total)
}

func TestPaymentInitiationsRetryCountsAttemptsAcrossPages(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := storage.NewMockStorage(ctrl)
	eng := engine.NewMockEngine(ctrl)
	s := New(store, eng, false)

	pid := models.PaymentInitiationID{Reference: "ref"}
	// 30 failed attempts, newest first: FAILED, PROCESSING, FAILED, PROCESSING, ...
	// followed by the original WAITING_FOR_VALIDATION adjustment.
	const failedAttempts = 30
	all := make([]models.PaymentInitiationAdjustment, 0, 2*failedAttempts+1)
	for i := 0; i < failedAttempts; i++ {
		all = append(all,
			models.PaymentInitiationAdjustment{Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED},
			models.PaymentInitiationAdjustment{Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING},
		)
	}
	all = append(all, models.PaymentInitiationAdjustment{Status: models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION})
	seen := map[uint64]int{}

	store.EXPECT().PaymentInitiationAdjustmentsList(gomock.Any(), pid, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ models.PaymentInitiationID, q storage.ListPaymentInitiationAdjustmentsQuery) (*paginate.Cursor[models.PaymentInitiationAdjustment], error) {
			from, to, hasNext, err := pagedOffsets(t, q.Offset, q.PageSize, len(all), seen)
			if err != nil {
				return nil, err
			}
			cursor := &paginate.Cursor[models.PaymentInitiationAdjustment]{PageSize: int(q.PageSize), HasMore: hasNext, Data: all[from:to]}
			if hasNext {
				next := q
				next.Offset = uint64(to)
				cursor.Next = paginate.EncodeCursor(next)
			}
			return cursor, nil
		}).AnyTimes()
	store.EXPECT().PaymentInitiationsGet(gomock.Any(), pid).Return(&models.PaymentInitiation{
		ID:   pid,
		Type: models.PAYMENT_INITIATION_TYPE_TRANSFER,
	}, nil)
	// The attempt number feeds the workflow ID, so it must count every
	// previous failure, not only the ones on the first page.
	eng.EXPECT().CreateTransfer(gomock.Any(), pid, failedAttempts+1, false).Return(models.Task{}, nil)

	_, err := s.PaymentInitiationsRetry(context.Background(), pid, false)
	require.NoError(t, err)
}
