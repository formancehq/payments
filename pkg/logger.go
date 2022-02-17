package payment

import (
	"github.com/numary/go-libs-cloud/pkg/sharedotlp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

type datadogAttributesAppender struct {
	underlying     logrus.Formatter
	serviceName    string
	env            string
	serviceVersion string
}

func (d datadogAttributesAppender) Format(entry *logrus.Entry) ([]byte, error) {
	span := trace.SpanFromContext(entry.Context)
	if !span.IsRecording() {
		return d.underlying.Format(entry)
	}
	return d.underlying.Format(entry.WithFields(logrus.Fields{
		"dd.trace_id": sharedotlp.ConvertToDatadogTraceId(span.SpanContext().TraceID().String()),
		"dd.span_id":  sharedotlp.ConvertToDatadogTraceId(span.SpanContext().SpanID().String()),
		"dd.service":  d.serviceName,
		"dd.env":      d.env,
		"dd.version":  d.serviceVersion,
	}))
}

var _ logrus.Formatter = &datadogAttributesAppender{}

func DatadogAttributesAppender(underlying logrus.Formatter, serviceName, env, serviceVersion string) *datadogAttributesAppender {
	return &datadogAttributesAppender{
		underlying:     underlying,
		serviceName:    serviceName,
		env:            env,
		serviceVersion: serviceVersion,
	}
}
