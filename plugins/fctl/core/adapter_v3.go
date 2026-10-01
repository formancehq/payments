package core

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	paymentscomponents "github.com/formancehq/payments/pkg/client/models/components"
)

func executeV3(ctx context.Context, request sdk.ExecuteRequest, host sdk.Host) error {
	command, spec, ok := commandByID(request.CommandID)
	if !ok {
		return invalid("unknown command %q", request.CommandID)
	}
	if err := sdk.ValidateExecuteRequest(command, request); err != nil {
		return invalid("invalid execution request: %v", err)
	}
	if err := sdk.ValidateTargetSelection(command.Target, request.Target); err != nil {
		return invalid("invalid target: %v", err)
	}
	arguments, err := collectArguments(command, request.Arguments)
	if err != nil {
		return err
	}
	flags, err := collectFlags(command, request.Flags)
	if err != nil {
		return err
	}
	if err := rejectFilteredCursor(command, flags); err != nil {
		return err
	}

	var response []byte
	var pageInfo *sdk.PageInfo
	for index, operation := range spec.operations {
		policy := command.Operations[index]
		priorResponse := response
		if operation.paginated && index == len(spec.operations)-1 {
			response, pageInfo, err = executePages(ctx, host, policy, operation, arguments, flags, request.Continuation, priorResponse)
		} else {
			response, _, err = executeOperation(ctx, host, policy, operation, arguments, flags, "", nil, priorResponse)
		}
		if err != nil {
			return fmt.Errorf("payments: %s: %w", operation.id, err)
		}
	}
	shape := sdk.ResultObject
	if spec.operations[len(spec.operations)-1].paginated {
		shape = sdk.ResultCollection
	}
	if len(response) == 0 {
		response = []byte(`{}`)
	}
	return host.Emit(sdk.Event{Kind: sdk.EventResult, Result: &sdk.ResultEnvelope{OperationID: command.ID, Shape: shape, MediaType: "application/json", Data: response, Page: pageInfo}})
}

// rejectFilteredCursor refuses a resume cursor alongside any other flag. A
// Payments cursor already encodes the filters and page size of the listing that
// produced it, so the product ignores a re-sent filter: accepting the
// combination would silently drop the caller's flag instead of honouring it.
func rejectFilteredCursor(command sdk.Command, flags map[string][]string) error {
	if first(flags["cursor"]) == "" {
		return nil
	}
	for _, declared := range command.Flags {
		if declared.Name == "cursor" || len(flags[declared.Name]) == 0 {
			continue
		}
		return invalid("cursor cannot be combined with %q: the cursor already encodes the original filters", declared.Name)
	}
	return nil
}

func executePages(ctx context.Context, host sdk.Host, policy sdk.OperationPolicy, operation operationSpec, arguments, flags map[string][]string, control sdk.ContinuationControl, priorResponse []byte) ([]byte, *sdk.PageInfo, error) {
	resume := first(flags["cursor"])
	if control.Mode != sdk.ContinuationAllPages {
		var firstPageBody []byte
		if resume != "" {
			firstPageBody = []byte{}
		}
		body, _, err := executeOperation(ctx, host, policy, operation, arguments, flags, resume, firstPageBody, priorResponse)
		if err != nil {
			return nil, nil, err
		}
		items, hasMore, next, err := parseCursorPage(body)
		if err != nil {
			return nil, nil, err
		}
		encoded, err := json.Marshal(items)
		if err != nil {
			return nil, nil, err
		}
		return encoded, &sdk.PageInfo{NextCursor: next, HasMore: hasMore}, nil
	}
	var bodyOverride []byte
	if handle := first(flags["query"]); handle != "" {
		var err error
		bodyOverride, err = readArtifact(ctx, host, handle)
		if err != nil {
			return nil, nil, err
		}
	}
	items := make([]json.RawMessage, 0)
	seen := map[string]struct{}{}
	cursor := resume
	if cursor != "" {
		seen[cursor] = struct{}{}
	}
	var aggregate uint64
	for page := uint32(0); page < control.MaxPages; page++ {
		pageBody := bodyOverride
		if cursor != "" {
			pageBody = []byte{}
		}
		body, next, err := executeOperation(ctx, host, policy, operation, arguments, flags, cursor, pageBody, priorResponse)
		if err != nil {
			return nil, nil, err
		}
		aggregate += uint64(len(body))
		if aggregate > control.MaxBytes {
			return nil, nil, sdk.Failure{Code: string(sdk.FailureInputTooLarge), Message: "paginated response exceeds aggregate byte limit"}
		}
		pageItems, hasMore, pageNext, err := parseCursorPage(body)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, pageItems...)
		if uint32(len(items)) > control.MaxItems {
			return nil, nil, sdk.Failure{Code: string(sdk.FailureInputTooLarge), Message: "paginated response exceeds item limit"}
		}
		if !hasMore {
			encoded, err := json.Marshal(items)
			return encoded, nil, err
		}
		if next == "" {
			next = pageNext
		}
		if next == "" {
			return nil, nil, invalid("paginated response hasMore without next cursor")
		}
		if _, exists := seen[next]; exists {
			return nil, nil, invalid("paginated response repeats a cursor")
		}
		seen[next] = struct{}{}
		cursor = next
	}
	return nil, nil, sdk.Failure{Code: string(sdk.FailureInputTooLarge), Message: "paginated response exceeds page limit"}
}

func parseCursorPage(body []byte) ([]json.RawMessage, bool, string, error) {
	var envelope struct {
		Cursor struct {
			Data    []json.RawMessage `json:"data"`
			HasMore bool              `json:"hasMore"`
			Next    string            `json:"next"`
		} `json:"cursor"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, false, "", invalid("invalid paginated response: %v", err)
	}
	if envelope.Cursor.Data == nil {
		envelope.Cursor.Data = []json.RawMessage{}
	}
	if envelope.Cursor.HasMore != (envelope.Cursor.Next != "") {
		return nil, false, "", invalid("paginated response cursor state is inconsistent")
	}
	return envelope.Cursor.Data, envelope.Cursor.HasMore, envelope.Cursor.Next, nil
}

func operationBody(ctx context.Context, host sdk.Host, operation operationSpec, arguments, flags map[string][]string) ([]byte, error) {
	if !operation.body {
		return nil, nil
	}
	if handle := first(arguments["input"]); handle != "" {
		return readArtifact(ctx, host, handle)
	}
	if handle := first(flags["query"]); handle != "" {
		return readArtifact(ctx, host, handle)
	}
	if operation.id == "v3ForwardBankAccount" {
		return json.Marshal(map[string]string{"connectorID": first(arguments["connector-id"])})
	}
	if operation.id == "v3UpdateBankAccountMetadata" || operation.id == "v3UpdatePaymentMetadata" {
		metadata := map[string]string{}
		for _, entry := range arguments["metadata"] {
			key, value, ok := strings.Cut(entry, "=")
			if !ok || key == "" {
				return nil, invalid("metadata must use key=value")
			}
			if _, exists := metadata[key]; exists {
				return nil, invalid("metadata key %q is repeated", key)
			}
			metadata[key] = value
		}
		return json.Marshal(struct {
			Metadata map[string]string `json:"metadata"`
		}{Metadata: metadata})
	}
	return nil, nil
}

func readArtifact(ctx context.Context, host sdk.Host, handle string) ([]byte, error) {
	var out []byte
	for {
		chunk, err := sdk.ReadInput(ctx, host, handle)
		if err != nil {
			return nil, err
		}
		if int64(len(out)+len(chunk.Bytes)) > requestBytes {
			return nil, sdk.Failure{Code: string(sdk.FailureInputTooLarge), Message: "input artifact is too large"}
		}
		out = append(out, chunk.Bytes...)
		if chunk.Final {
			break
		}
		if len(chunk.Bytes) == 0 {
			return nil, invalid("input artifact made no progress")
		}
	}
	if !json.Valid(out) {
		return nil, invalid("input artifact must contain JSON")
	}
	return out, nil
}

func commandByID(id string) (sdk.Command, commandSpec, bool) {
	commands, specs := Catalogue(), catalogueSpecs()
	for index := range commands {
		if commands[index].ID == id {
			return commands[index], specs[index], true
		}
	}
	return sdk.Command{}, commandSpec{}, false
}

func collectArguments(command sdk.Command, values []string) (map[string][]string, error) {
	out := map[string][]string{}
	position := 0
	for _, argument := range command.Arguments {
		if argument.Repeated {
			if argument.Required && position == len(values) {
				return nil, invalid("missing required argument %q", argument.Name)
			}
			out[argument.Name] = append([]string(nil), values[position:]...)
			position = len(values)
			continue
		}
		if position < len(values) {
			out[argument.Name] = []string{values[position]}
			position++
		} else if argument.Required {
			return nil, invalid("missing required argument %q", argument.Name)
		}
	}
	if position != len(values) {
		return nil, invalid("too many arguments")
	}
	return out, nil
}

func collectFlags(command sdk.Command, values []sdk.FlagOccurrence) (map[string][]string, error) {
	declared := map[string]sdk.Flag{}
	for _, value := range command.Flags {
		declared[value.Name] = value
	}
	out := map[string][]string{}
	for _, value := range values {
		spec, ok := declared[value.Name]
		if !ok {
			return nil, invalid("unknown flag %q", value.Name)
		}
		if spec.Type != sdk.FlagStringArray && len(out[value.Name]) > 0 {
			return nil, invalid("flag %q is repeated", value.Name)
		}
		out[value.Name] = append(out[value.Name], value.Value)
	}
	for _, value := range command.Flags {
		if value.Required && len(out[value.Name]) == 0 {
			return nil, invalid("missing required flag %q", value.Name)
		}
	}
	return out, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
func invalid(format string, values ...any) error {
	return sdk.Failure{Code: string(sdk.FailureInvalidArgument), Message: fmt.Sprintf(format, values...)}
}

// credentialFieldNames is the connector-configuration redaction denylist. It is
// exhaustive by construction: connector_secrets_test.go fails when a
// regenerated client introduces a configuration property that appears neither
// here nor in the reviewed non-credential acknowledgement list.
var credentialFieldNames = map[string]struct{}{
	"accessKey": {}, "apiKey": {}, "apiSecret": {}, "clientSecret": {},
	"configurationToken": {}, "passphrase": {}, "password": {}, "privateKey": {},
	"secret": {}, "stagingToken": {}, "userCertificate": {}, "userCertificateKey": {},
	"webhookPassword": {}, "webhookSharedSecret": {},
}

func isCredentialFieldName(name string) bool {
	_, sensitive := credentialFieldNames[name]
	return sensitive
}

func redactConnectorConfig(encoded []byte) ([]byte, error) {
	var response paymentscomponents.V3GetConnectorConfigResponse
	if err := json.Unmarshal(encoded, &response); err != nil {
		return nil, invalid("invalid connector config response")
	}
	var walk func(reflect.Value)
	walk = func(current reflect.Value) {
		if !current.IsValid() {
			return
		}
		if current.Kind() == reflect.Pointer {
			if current.IsNil() {
				return
			}
			walk(current.Elem())
			return
		}
		if current.Kind() != reflect.Struct {
			return
		}
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			metadata := current.Type().Field(index)
			key := strings.Split(metadata.Tag.Get("json"), ",")[0]
			if isCredentialFieldName(key) {
				switch field.Kind() {
				case reflect.String:
					if field.CanSet() {
						field.SetString("[REDACTED]")
					}
				case reflect.Pointer:
					if !field.IsNil() && field.Type().Elem().Kind() == reflect.String && field.CanSet() {
						redacted := reflect.New(field.Type().Elem())
						redacted.Elem().SetString("[REDACTED]")
						field.Set(redacted)
					}
				}
				continue
			}
			walk(field)
		}
	}
	walk(reflect.ValueOf(&response))
	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("payments: encode redacted connector config: %w", err)
	}
	return result, nil
}
