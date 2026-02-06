package pagination

import "github.com/formancehq/payments/pkg/connector"

// ShouldFetchMore is an alias to pkg/connector.ShouldFetchMore for backward compatibility.
// The canonical implementation now lives in pkg/connector.
func ShouldFetchMore[T any, C any](total []T, currentBatch []C, pageSize int) (needMore bool, hasMore bool) {
	return connector.ShouldFetchMore(total, currentBatch, pageSize)
}
