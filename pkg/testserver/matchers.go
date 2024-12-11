package testserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/formancehq/go-libs/v2/publish"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/go-cmp/cmp"
	"github.com/invopop/jsonschema"
	"github.com/nats-io/nats.go"
	"github.com/onsi/gomega/types"
	"github.com/xeipuuv/gojsonschema"
)

type PayloadMatcher interface {
	Match(actual interface{}) error
}

type NoOpPayloadMatcher struct{}

func (n NoOpPayloadMatcher) Match(interface{}) error {
	return nil
}

var _ PayloadMatcher = (*NoOpPayloadMatcher)(nil)

type CallbackMatcher struct {
	expected any
	callback func(b []byte) error
}

func (m CallbackMatcher) Match(payload interface{}) error {
	marshaledPayload, err := rawJson(m.expected, payload, true)
	if err != nil {
		return fmt.Errorf("failed to process payload: %s", err)
	}
	return m.callback(marshaledPayload)
}

func WithCallback(expected any, callback func(b []byte) error) CallbackMatcher {
	return CallbackMatcher{
		expected: expected,
		callback: callback,
	}
}

var _ PayloadMatcher = (*CallbackMatcher)(nil)

type StructPayloadMatcher struct {
	expected any
	strict   bool
}

func (s StructPayloadMatcher) Match(payload interface{}) error {
	marshaledPayload, err := rawJson(s.expected, payload, s.strict)
	if err != nil {
		return fmt.Errorf("failed to process payload: %s", err)
	}

	unmarshalledPayload := reflect.New(reflect.TypeOf(s.expected)).Interface()
	if err := json.Unmarshal(marshaledPayload, unmarshalledPayload); err != nil {
		return fmt.Errorf("unable to unmarshal payload: %s", err)
	}

	// unmarshalledPayload is actually a pointer
	// as it is seen as "any" by the code, we use reflection to get the targeted valud
	unmarshalledPayload = reflect.ValueOf(unmarshalledPayload).Elem().Interface()

	diff := cmp.Diff(unmarshalledPayload, s.expected, cmp.Comparer(func(v1 *big.Int, v2 *big.Int) bool {
		return v1.String() == v2.String()
	}))
	if diff != "" {
		return errors.New(diff)
	}

	return nil
}

func WithPayload(v any) StructPayloadMatcher {
	return StructPayloadMatcher{
		expected: v,
		strict:   true,
	}
}

// WithPayloadSubset is able to match partial structs
func WithPayloadSubset(v any) StructPayloadMatcher {
	return StructPayloadMatcher{
		expected: v,
		strict:   false,
	}
}

var _ PayloadMatcher = (*StructPayloadMatcher)(nil)

// todo(libs): move in shared libs
type EventMatcher struct {
	eventName string
	matchers  []PayloadMatcher
	err       error
}

func (e *EventMatcher) Match(actual any) (success bool, err error) {
	msg, ok := actual.(*nats.Msg)
	if !ok {
		return false, fmt.Errorf("expected type %t", actual)
	}

	ev := publish.EventMessage{}
	if err := json.Unmarshal(msg.Data, &ev); err != nil {
		return false, fmt.Errorf("unable to unmarshal msg: %s", err)
	}

	if ev.Type != e.eventName {
		return false, nil
	}

	for _, matcher := range e.matchers {
		if e.err = matcher.Match(ev.Payload); e.err != nil {
			return false, nil
		}
	}

	return true, nil
}

func (e *EventMatcher) FailureMessage(_ any) (message string) {
	return fmt.Sprintf("event does not match expectations: %s", e.err)
}

func (e *EventMatcher) NegatedFailureMessage(_ any) (message string) {
	return "event should not match"
}

var _ types.GomegaMatcher = (*EventMatcher)(nil)

func Event(eventName string, matchers ...PayloadMatcher) types.GomegaMatcher {
	return &EventMatcher{
		matchers:  matchers,
		eventName: eventName,
	}
}

type TaskMatcher struct {
	status   models.TaskStatus
	matchers []PayloadMatcher
	err      error
}

func (t *TaskMatcher) Match(actual any) (success bool, err error) {
	task, ok := actual.(models.Task)
	if !ok {
		return false, fmt.Errorf("unexpected type %t", actual)
	}

	if task.Status != t.status {
		return false, nil
	}

	for _, matcher := range t.matchers {
		if t.err = matcher.Match(task); t.err != nil {
			return false, nil
		}
	}
	return true, nil
}

func (t *TaskMatcher) FailureMessage(_ any) (message string) {
	return fmt.Sprintf("event does not match expectations: %s", t.err)
}

func (t *TaskMatcher) NegatedFailureMessage(_ any) (message string) {
	return "event should not match"
}

var _ types.GomegaMatcher = (*TaskMatcher)(nil)

func HaveTaskStatus(status models.TaskStatus, matchers ...PayloadMatcher) types.GomegaMatcher {
	return &TaskMatcher{
		matchers: matchers,
		status:   status,
	}
}

func rawJson(expected any, payload interface{}, strict bool) (b []byte, err error) {
	rawSchema := jsonschema.Reflect(expected)
	data, err := json.Marshal(rawSchema)
	if err != nil {
		return b, fmt.Errorf("unable to marshal schema: %s", err)
	}

	schemaJSONLoader := gojsonschema.NewStringLoader(string(data))
	schema, err := gojsonschema.NewSchema(schemaJSONLoader)
	if err != nil {
		return b, fmt.Errorf("unable to load json schema: %s", err)
	}

	dataJsonLoader := gojsonschema.NewRawLoader(payload)

	if strict {
		validate, err := schema.Validate(dataJsonLoader)
		if err != nil {
			return b, fmt.Errorf("failed to validate: %w", err)
		}

		if !validate.Valid() {
			return b, fmt.Errorf("validation errors: %s", validate.Errors())
		}
	}

	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return b, fmt.Errorf("unable to marshal payload: %s", err)
	}
	return marshaledPayload, nil
}
