package payment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/propagation"
)

const (
	TopicSavedPayment = "payment.saved"
)

type SavedPaymentEvent struct {
	Payment Payment `json:"payment"`
}

func newMessage(ctx context.Context, v interface{}) *message.Message {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	msg := message.NewMessage(watermill.NewUUID(), data)

	p := propagation.TraceContext{}
	carrier := propagation.MapCarrier{}
	p.Inject(ctx, carrier)

	data, err = json.Marshal(carrier)
	if err != nil {
		panic(err)
	}
	msg.Metadata.Set("tracing-context", base64.StdEncoding.EncodeToString(data))
	msg.SetContext(ctx)
	return msg
}
