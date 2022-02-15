package payment

import (
	"encoding/json"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	TopicCreatedPayment = "payment.created"
	TopicUpdatedPayment = "payment.updated"
)

type CreatedPaymentEvent struct {
	Payment Payment `json:"payment"`
}

type UpdatedPaymentEvent struct {
	ID   string `json:"id"`
	Data Data   `json:"data"`
}

func newMessage(v interface{}) *message.Message {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return message.NewMessage(watermill.NewUUID(), data)
}
