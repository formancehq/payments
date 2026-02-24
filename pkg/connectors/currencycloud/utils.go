package currencycloud

func shouldFetchMore[T any](ret []T, nextPage, pageSize int) (bool, bool, []T) {
	switch {
	case len(ret) > pageSize:
		return false, true, ret[:pageSize]
	case len(ret) == pageSize:
		if nextPage != -1 {
			// more pages, but we have enough
			return false, true, ret
		}
		return false, false, ret
	case nextPage == -1:
		// No more pages
		return false, false, ret
	}
	return true, true, ret
}
