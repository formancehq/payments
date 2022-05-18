package http

import (
	"net/http"
	"strconv"
	"strings"
)

func Bool(r *http.Request, key string) (bool, bool) {
	vv := r.URL.Query().Get(key)
	if vv == "" {
		return false, false
	}
	vv = strings.ToUpper(vv)
	return vv == "YES" || vv == "TRUE" || vv == "1", true
}

func Integer(r *http.Request, key string) (int64, bool, error) {
	if value := r.URL.Query().Get(key); value != "" {
		ret, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false, err
		}
		return ret, true, nil
	}
	return 0, false, nil
}

func IntegerWithDefault(r *http.Request, key string, def int64) (int64, error) {
	value, ok, err := Integer(r, key)
	if err != nil {
		return 0, err
	}
	if ok {
		return value, nil
	}
	return def, nil
}
