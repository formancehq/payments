package otel

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	once   sync.Once
	tracer trace.Tracer
)

func Tracer() trace.Tracer {
	once.Do(func() {
		tracer = otel.Tracer("com.formance.payments")
	})

	return tracer
}

func RecordError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func StartSpan(
	ctx context.Context,
	spanName string,
	attributes ...attribute.KeyValue,
) (context.Context, trace.Span) {
	parentSpan := trace.SpanFromContext(ctx)
	return Tracer().Start(
		ctx,
		spanName,
		trace.WithNewRoot(),
		trace.WithLinks(trace.Link{
			SpanContext: parentSpan.SpanContext(),
		}),
		trace.WithAttributes(
			attributes...,
		),
	)
}
