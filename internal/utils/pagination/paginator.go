package pagination

// ShouldFetchMore determines if we need more objects to fetch and if we have
// more objects to fetch.
// This function should be used only if the following requirements are met:
// - When fetching some objects, you need to append all of them to a slice
func ShouldFetchMore[T any, C any](total []T, currentBatch []C, pageSize int) (needMore bool, hasMore bool) {
	switch {
	case len(total) > pageSize:
		// We fetched more than we should, the total will be trimed, we don't
		// need more
		needMore = false
		// Since the total will be trimed it means we will have to refetch the
		// objects trimed, so we have more
		hasMore = true
	case len(total) == pageSize:
		// We don't need more, we fetched exactly what we needed
		needMore = false
		// hasMore depennds on the currentBatch, if the currentBatch is full
		// then we have more
		hasMore = len(currentBatch) >= pageSize
	default:
		// Here, total is < pageSize, so we need more objects
		needMore = true
		// If the currentBatch is full, we have more
		hasMore = len(currentBatch) >= pageSize
	}
	return
}
