package storage

import "github.com/uptrace/bun"

type Storage struct {
	db *bun.DB
}

func newStorage(db *bun.DB) *Storage {
	return &Storage{db: db}
}
