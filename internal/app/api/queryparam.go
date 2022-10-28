package api

import (
	"net/http"
	"strconv"
)

func integer(r *http.Request, key string) (int64, bool, error) {
	if value := r.URL.Query().Get(key); value != "" {
		ret, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false, err
		}

		return ret, true, nil
	}

	return 0, false, nil
}

func integerWithDefault(r *http.Request, key string, def int64) (int64, error) {
	value, ok, err := integer(r, key)
	if err != nil {
		return 0, err
	}

	if ok {
		return value, nil
	}

	return def, nil
}
