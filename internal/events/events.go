package events

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/formancehq/go-libs/v3/publish"
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

func (e *Events) Publish(ctx context.Context, em publish.EventMessage) error {
	return e.publisher.Publish(eventsdef.TopicPayments,
		publish.NewMessage(ctx, em))
}

func bigIntToString(v *big.Int) *string {
	if v == nil {
		return nil
	}
	s := v.String()
	return &s
}

func bigIntFromString(s *string, field string) (*big.Int, error) {
	if s == nil {
		return nil, nil
	}
	bi := new(big.Int)
	if _, ok := bi.SetString(*s, 10); !ok {
		return nil, fmt.Errorf("invalid %s string: %s", field, *s)
	}
	return bi, nil
}
