package services

import (
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/storage"
)

type Service struct {
	storage storage.Storage
	engine  engine.Engine
	debug   bool
}

func New(storage storage.Storage, engine engine.Engine, debug bool) *Service {
	return &Service{
		storage: storage,
		engine:  engine,
		debug:   debug,
	}
}

var _ backend.Backend = &Service{}
