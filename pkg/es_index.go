package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	_ "github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"time"
)

type Record struct {
	Kind   string      `json:"kind"`
	Ledger string      `json:"ledger"`
	When   time.Time   `json:"when"`
	Data   interface{} `json:"data"`
}

func insert(ctx context.Context, index string, t esapi.Transport, payment Payment) error {
	data, err := json.Marshal(Record{
		Kind: "ACCOUNT",
		When: time.Now(),
		Data: payment,
	})
	if err != nil {
		return errors.Wrapf(err, "encoding payment '%s'", payment.ID)
	}

	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: payment.ID,
		Refresh:    "true",
		Body:       bytes.NewReader(data),
		Pipeline:   "PAYMENT",
	}
	res, err := req.Do(ctx, t)
	if err != nil {
		return errors.Wrapf(err, "error making request to es")
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error inserting es index: %s [%s]", res.Status(), res.String())
	}

	if res.HasWarnings() {
		for _, w := range res.Warnings() {
			logrus.Warn(w)
		}
	}
	return nil
}

func ReplicatePaymentOnES(ctx context.Context, subscriber message.Subscriber, index string, t esapi.Transport) {
	createdPayments, err := subscriber.Subscribe(ctx, TopicCreatedPayment)
	if err != nil {
		panic(err)
	}
	updatedPayments, err := subscriber.Subscribe(ctx, TopicUpdatedPayment)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case createdPayment := <-createdPayments:
			event := CreatedPaymentEvent{}
			err := json.Unmarshal(createdPayment.Payload, &event)
			if err != nil {
				logrus.Errorf("error processing message '%s': %s", createdPayment.UUID, err)
				continue
			}

			err = insert(ctx, index, t, event.Payment)
			if err != nil {
				logrus.Errorf("error inserting payment on es: %s", err)
				continue
			}

			createdPayment.Ack()

		case updatedPayment := <-updatedPayments:
			event := UpdatedPaymentEvent{}
			err := json.Unmarshal(updatedPayment.Payload, &event)
			if err != nil {
				logrus.Errorf("error processing message '%s': %s", updatedPayment.UUID, err)
				continue
			}

			err = insert(ctx, index, t, Payment{
				Data: event.Data,
				ID:   event.ID,
			})
			if err != nil {
				logrus.Errorf("error updating payment on es: %s", err)
				continue
			}

			updatedPayment.Ack()
		case <-ctx.Done():
			return
		}
	}
}
