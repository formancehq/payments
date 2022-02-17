package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	_ "github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/numary/go-libs-cloud/pkg/sharedotlp"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

	//extractCtx := func(msg *message.Message) context.Context {
	//	tracingContext := msg.Metadata.Get("tracing-context")
	//	data, err := base64.StdEncoding.DecodeString(tracingContext)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	carrier := propagation.MapCarrier{}
	//	err = json.Unmarshal(data, &carrier)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	p := propagation.TraceContext{}
	//	return p.Extract(ctx, carrier)
	//}

	for {
		select {
		case createdPayment := <-createdPayments:
			createdPayment.Ack()
			func() {
				ctx, span := otel.Tracer("com.numary.payments.indexer",
					trace.WithInstrumentationVersion("semver:1.0.0")).Start(ctx, "Event.Created",
					trace.WithSpanKind(trace.SpanKindClient),
					trace.WithAttributes(attribute.String("resource.name", "Event.Created"), attribute.String("span.name", "Event.Created")) /*, trace.WithLinks(trace.LinkFromContext(extractCtx(createdPayment)))*/)
				defer span.End()
				defer sharedotlp.RecordErrorOnRecover(ctx, false)()

				event := CreatedPaymentEvent{}
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

		case updatedPayment := <-updatedPayments:
			updatedPayment.Ack()
			func() {

				ctx, span := otel.Tracer("com.numary.payments.indexer",
					trace.WithInstrumentationVersion("semver:1.0.0")).Start(ctx, "Event.Updated" /*, trace.WithLinks(trace.LinkFromContext(extractCtx(updatedPayment)))*/)
				defer span.End()
				defer sharedotlp.RecordErrorOnRecover(ctx, false)()

				event := UpdatedPaymentEvent{}
				err := json.Unmarshal(updatedPayment.Payload, &event)
				if err != nil {
					sharedotlp.RecordError(ctx, err)
					return
				}

				err = insert(ctx, index, t, Payment{
					Data: event.Data,
					ID:   event.ID,
				})
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
