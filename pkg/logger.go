package payment

import (
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"strconv"
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
		"dd.trace_id": convertTraceID(span.SpanContext().TraceID().String()),
		"dd.span_id":  convertTraceID(span.SpanContext().SpanID().String()),
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

// See https://docs.datadoghq.com/fr/tracing/connect_logs_and_traces/opentelemetry/
func convertTraceID(id string) string {
	if len(id) < 16 {
		return ""
	}
	if len(id) > 16 {
		id = id[16:]
	}
	intValue, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(intValue, 10)
}
