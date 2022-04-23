package payment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/propagation"
	"time"
)

const (
	TopicSavedPayment = "SAVED_PAYMENT"
)

type SavedPaymentEvent struct {
	Date    time.Time `json:"date"`
	Type    string    `json:"type"`
	Payload Payment   `json:"payload"`
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
