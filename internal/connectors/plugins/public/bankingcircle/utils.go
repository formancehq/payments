package bankingcircle

func shouldFetchMore[T any](ret []T, pageSize int) (bool, bool, []T) {
	switch {
	case len(ret) >= pageSize:
		return false, true, ret[:pageSize]
	default:
		return true, true, ret
	}
}
