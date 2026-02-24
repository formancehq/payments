package adyen

func shouldFetchMore[T any, V any](ret []T, pagedT []V, pageSize int) (bool, bool, []T) {
	switch {
	case len(pagedT) < pageSize:
		return false, false, ret
	case len(ret) >= pageSize:
		return false, true, ret[:pageSize]
	default:
		return true, true, ret
	}
}
