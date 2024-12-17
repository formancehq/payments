package models

import (
	//nolint:gosec
	"crypto/sha1"
	"fmt"

	"github.com/gibson042/canonicaljson-go"
)

func IdempotencyKey(u any) string {
	data, err := canonicaljson.Marshal(u)
	if err != nil {
		panic(err)
	}
	//nolint:gosec
	hash := sha1.Sum(data)
	return fmt.Sprintf("%x", hash)
}
