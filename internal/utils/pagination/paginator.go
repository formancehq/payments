package pagination

func ShouldFetchMore[T any, C any](total []T, currentBatch []C, pageSize int) (bool, bool) {
	return len(total) < pageSize, len(currentBatch) >= pageSize
}
