package services

import (
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/events"
	"github.com/formancehq/payments/internal/storage"
)

type Service struct {
	storage storage.Storage

	engine engine.Engine
	events *events.Events
}

func New(storage storage.Storage, engine engine.Engine, events *events.Events) *Service {
	return &Service{
		storage: storage,
		engine:  engine,
		events:  events,
	}
}
