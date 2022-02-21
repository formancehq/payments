package payment

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/numary/go-libs-cloud/pkg/sharedotlp"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"time"
)

type Record struct {
	Kind string      `json:"kind"`
	When time.Time   `json:"when"`
	Data interface{} `json:"data"`
}

func insert(ctx context.Context, index string, t esapi.Transport, payment Payment) error {
	data, err := json.Marshal(Record{
		Kind: "PAYMENT",
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
	payments, err := subscriber.Subscribe(ctx, TopicSavedPayment)
	if err != nil {
		panic(err)
	}

	extractCtx := func(msg *message.Message) context.Context {
		tracingContext := msg.Metadata.Get("tracing-context")
		data, err := base64.StdEncoding.DecodeString(tracingContext)
		if err != nil {
			panic(err)
		}

		carrier := propagation.MapCarrier{}
		err = json.Unmarshal(data, &carrier)
		if err != nil {
			panic(err)
		}

		p := propagation.TraceContext{}
		return p.Extract(ctx, carrier)
	}

	tracer := otel.Tracer("com.numary.payments.indexer")

	for {
		select {
		case createdPayment := <-payments:
			createdPayment.Ack()
			func() {
				ctx, span := tracer.Start(
					context.Background(),
					"Event.Created",
					trace.WithSpanKind(trace.SpanKindClient),
					trace.WithLinks(trace.LinkFromContext(extractCtx(createdPayment))),
				)
				defer span.End()
				defer sharedotlp.RecordErrorOnRecover(ctx, false)()

				event := SavedPaymentEvent{}
				err = json.Unmarshal(createdPayment.Payload, &event)
				if err != nil {
					sharedotlp.RecordError(ctx, err)
					return
				}

				err = insert(ctx, index, t, event.Payment)
				if err != nil {
					sharedotlp.RecordError(ctx, err)
					return
				}
			}()
		case <-ctx.Done():
			return
		}
	}
}
