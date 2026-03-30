package client

import (
	"fmt"
	"net/url"
	"strconv"
)

var queryTemplate = `{"$gt":{"importedAt":"%s"}}`

func RawQuery(v url.Values, pageSize int, cursor string, importedAt string) string {
	v.Add("pageSize", strconv.Itoa(pageSize))

	// rely on cursor if present
	if cursor != "" {
		v.Add("cursor", cursor)
	}

	// in cases where we had less than a page we no longer have a cursor: so we rely on the last known
	// import time to avoid starting again from the beginning of time
	if cursor == "" && importedAt != "" {
		queryStr := fmt.Sprintf(queryTemplate, importedAt)
		v.Add("query", queryStr)
	}
	return v.Encode()
}
