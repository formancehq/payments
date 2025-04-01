package events

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v2/publish"
	eventsdef "github.com/formancehq/payments/pkg/events"
)

type Events struct {
	publisher message.Publisher

	stackURL string
}

func New(p message.Publisher, stackURL string) *Events {
	return &Events{
		publisher: p,
		stackURL:  stackURL,
	}
}

func (e *Events) Publish(ctx context.Context, em ...publish.EventMessage) error {
	messages := make([]*message.Message, 0, len(em))
	for _, e := range em {
		messages = append(messages, publish.NewMessage(ctx, e))
	}
	return e.publisher.Publish(eventsdef.TopicPayments, messages...)
}
