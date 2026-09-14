package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk/producthttp"
	paymentsclient "github.com/formancehq/payments/pkg/client"
	paymentscomponents "github.com/formancehq/payments/pkg/client/models/components"
	paymentsoperations "github.com/formancehq/payments/pkg/client/models/operations"
)

const generatedClientBaseURL = "https://product.invalid"

// capturedProductClient lets the generated SDK consume and validate its typed
// response while retaining the exact bounded product bytes emitted to fctl.
type capturedProductClient struct {
	client *producthttp.Client
	body   []byte
}

func (client *capturedProductClient) Do(request *http.Request) (*http.Response, error) {
	client.body = nil
	response, err := client.client.Do(request)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("payments generated client: read response: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("payments generated client: close response: %w", closeErr)
	}
	client.body = append([]byte(nil), body...)
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	return response, nil
}

func executeOperation(ctx context.Context, host sdk.Host, policy sdk.OperationPolicy, operation operationSpec, arguments, flags map[string][]string, cursor string, bodyOverride, priorResponse []byte) ([]byte, string, error) {
	body := bodyOverride
	if body == nil {
		var err error
		body, err = operationBody(ctx, host, operation, arguments, flags)
		if err != nil {
			return nil, "", err
		}
	}

	bridge, err := producthttp.New(host, policy, "auth.stack")
	if err != nil {
		return nil, "", err
	}
	captured := &capturedProductClient{client: bridge}
	generated := paymentsclient.New(
		generatedClientBaseURL,
		paymentsclient.WithClient(captured),
		paymentsclient.WithSecuritySource(func(context.Context) (paymentscomponents.Security, error) {
			return paymentscomponents.Security{}, nil
		}),
	)
	if err := invokeGeneratedV3(ctx, generated.Payments.V3, operation, arguments, flags, cursor, body, priorResponse); err != nil {
		if operation.id == "v3GetConnectorConfig" && len(captured.body) != 0 {
			return nil, "", invalid("invalid connector config response")
		}
		return nil, "", err
	}
	if operation.id == "v3GetConnectorConfig" {
		redacted, err := redactConnectorConfig(captured.body)
		if err != nil {
			return nil, "", err
		}
		return redacted, "", nil
	}
	return append([]byte(nil), captured.body...), "", nil
}

func invokeGeneratedV3(ctx context.Context, client *paymentsclient.V3, operation operationSpec, arguments, flags map[string][]string, cursor string, body, priorResponse []byte) error {
	pageSize, err := generatedPageSize(flags, cursor)
	if err != nil {
		return err
	}
	cursorValue := optionalString(cursor)
	requestMap, err := generatedRequestMap(body)
	if err != nil {
		return err
	}

	switch operation.id {
	case "v3CreateAccount":
		request, err := generatedBody[paymentscomponents.V3CreateAccountRequest](body)
		if err != nil {
			return err
		}
		_, err = client.CreateAccount(ctx, request)
		return err
	case "v3ListAccounts":
		_, err = client.ListAccounts(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3GetAccount":
		_, err = client.GetAccount(ctx, first(arguments["account-id"]))
		return err
	case "v3GetAccountBalances":
		var asset *string
		var from, to *time.Time
		if cursor == "" {
			asset = optionalString(first(flags["asset"]))
			from, err = optionalTime(flags, "from-timestamp")
			if err != nil {
				return err
			}
			to, err = optionalTime(flags, "to-timestamp")
			if err != nil {
				return err
			}
		}
		_, err = client.GetAccountBalances(ctx, paymentsoperations.V3GetAccountBalancesRequest{
			AccountID: first(arguments["account-id"]), Asset: asset, FromTimestamp: from, ToTimestamp: to, PageSize: pageSize, Cursor: cursorValue,
		})
		return err
	case "v3CreateBankAccount":
		request, err := generatedBody[paymentscomponents.V3CreateBankAccountRequest](body)
		if err != nil {
			return err
		}
		_, err = client.CreateBankAccount(ctx, request)
		return err
	case "v3ListBankAccounts":
		_, err = client.ListBankAccounts(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3GetBankAccount":
		_, err = client.GetBankAccount(ctx, first(arguments["bank-account-id"]))
		return err
	case "v3UpdateBankAccountMetadata":
		request, err := generatedBody[paymentscomponents.V3UpdateBankAccountMetadataRequest](body)
		if err != nil {
			return err
		}
		_, err = client.UpdateBankAccountMetadata(ctx, first(arguments["bank-account-id"]), request)
		return err
	case "v3ForwardBankAccount":
		request, err := generatedBody[paymentscomponents.V3ForwardBankAccountRequest](body)
		if err != nil {
			return err
		}
		_, err = client.ForwardBankAccount(ctx, first(arguments["bank-account-id"]), request)
		return err
	case "v3ListConnectors":
		_, err = client.ListConnectors(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3InstallConnector":
		body, canonicalConnector, err := generatedConnectorBody(body, first(arguments["connector"]), priorResponse)
		if err != nil {
			return err
		}
		request, err := generatedBody[paymentscomponents.V3InstallConnectorRequest](body)
		if err != nil {
			return err
		}
		_, err = client.InstallConnector(ctx, strings.ToLower(canonicalConnector), request)
		return err
	case "v3ListConnectorConfigs":
		_, err = client.ListConnectorConfigs(ctx)
		return err
	case "v3UninstallConnector":
		_, err = client.UninstallConnector(ctx, first(flags["connector-id"]))
		return err
	case "v3GetConnectorConfig":
		_, err = client.GetConnectorConfig(ctx, first(flags["connector-id"]))
		return err
	case "v3UpdateConnectorConfig":
		body, _, err := generatedConnectorBody(body, first(arguments["connector"]), priorResponse)
		if err != nil {
			return err
		}
		request, err := generatedBody[paymentscomponents.V3UpdateConnectorRequest](body)
		if err != nil {
			return err
		}
		_, err = client.V3UpdateConnectorConfig(ctx, first(flags["connector-id"]), request)
		return err
	case "v3ListConnectorSchedules":
		_, err = client.ListConnectorSchedules(ctx, first(arguments["connector-id"]), pageSize, cursorValue, requestMap)
		return err
	case "v3GetConnectorSchedule":
		_, err = client.GetConnectorSchedule(ctx, first(arguments["connector-id"]), first(arguments["schedule-id"]))
		return err
	case "v3ListConnectorScheduleInstances":
		_, err = client.ListConnectorScheduleInstances(ctx, first(arguments["connector-id"]), first(arguments["schedule-id"]), pageSize, cursorValue)
		return err
	case "v3ListOrders":
		_, err = client.ListOrders(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3GetOrder":
		_, err = client.GetOrder(ctx, first(arguments["order-id"]))
		return err
	case "v3ListConversions":
		_, err = client.ListConversions(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3GetConversion":
		_, err = client.GetConversion(ctx, first(arguments["conversion-id"]))
		return err
	case "v3CreatePayment":
		request, err := generatedBody[paymentscomponents.V3CreatePaymentRequest](body)
		if err != nil {
			return err
		}
		_, err = client.CreatePayment(ctx, request)
		return err
	case "v3ListPayments":
		_, err = client.ListPayments(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3GetPayment":
		_, err = client.GetPayment(ctx, first(arguments["payment-id"]))
		return err
	case "v3UpdatePaymentMetadata":
		request, err := generatedBody[paymentscomponents.V3UpdatePaymentMetadataRequest](body)
		if err != nil {
			return err
		}
		_, err = client.UpdatePaymentMetadata(ctx, first(arguments["payment-id"]), request)
		return err
	case "v3InitiatePayment":
		request, err := generatedBody[paymentscomponents.V3InitiatePaymentRequest](body)
		if err != nil {
			return err
		}
		noValidation, err := optionalBool(flags, "no-validation")
		if err != nil {
			return err
		}
		_, err = client.InitiatePayment(ctx, noValidation, request)
		return err
	case "v3ListPaymentInitiations":
		_, err = client.ListPaymentInitiations(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3DeletePaymentInitiation":
		_, err = client.DeletePaymentInitiation(ctx, first(arguments["transfer-id"]))
		return err
	case "v3GetPaymentInitiation":
		_, err = client.GetPaymentInitiation(ctx, first(arguments["transfer-id"]))
		return err
	case "v3RetryPaymentInitiation":
		_, err = client.RetryPaymentInitiation(ctx, first(arguments["transfer-id"]))
		return err
	case "v3ApprovePaymentInitiation":
		_, err = client.ApprovePaymentInitiation(ctx, first(arguments["transfer-id"]))
		return err
	case "v3RejectPaymentInitiation":
		_, err = client.RejectPaymentInitiation(ctx, first(arguments["transfer-id"]))
		return err
	case "v3ReversePaymentInitiation":
		request, err := generatedBody[paymentscomponents.V3ReversePaymentInitiationRequest](body)
		if err != nil {
			return err
		}
		_, err = client.ReversePaymentInitiation(ctx, first(arguments["transfer-id"]), request)
		return err
	case "v3CreatePool":
		request, err := generatedBody[paymentscomponents.V3CreatePoolRequest](body)
		if err != nil {
			return err
		}
		_, err = client.CreatePool(ctx, request)
		return err
	case "v3ListPools":
		_, err = client.ListPools(ctx, pageSize, cursorValue, requestMap)
		return err
	case "v3GetPool":
		_, err = client.GetPool(ctx, first(arguments["pool-id"]))
		return err
	case "v3DeletePool":
		_, err = client.DeletePool(ctx, first(arguments["pool-id"]))
		return err
	case "v3UpdatePoolQuery":
		request, err := generatedBody[paymentscomponents.V3UpdatePoolQueryRequest](body)
		if err != nil {
			return err
		}
		_, err = client.UpdatePoolQuery(ctx, first(arguments["pool-id"]), request)
		return err
	case "v3GetPoolBalances":
		at, err := requiredTime(arguments, "at")
		if err != nil {
			return err
		}
		_, err = client.GetPoolBalances(ctx, first(arguments["pool-id"]), &at)
		return err
	case "v3GetPoolBalancesLatest":
		_, err = client.GetPoolBalancesLatest(ctx, first(arguments["pool-id"]))
		return err
	case "v3AddAccountToPool":
		_, err = client.AddAccountToPool(ctx, first(arguments["pool-id"]), first(arguments["account-id"]))
		return err
	case "v3RemoveAccountFromPool":
		_, err = client.RemoveAccountFromPool(ctx, first(arguments["pool-id"]), first(arguments["account-id"]))
		return err
	case "v3GetTask":
		_, err = client.GetTask(ctx, first(arguments["task-id"]))
		return err
	default:
		return invalid("operation %q has no generated client mapping", operation.id)
	}
}

func generatedBody[T any](body []byte) (*T, error) {
	if len(body) == 0 {
		return nil, invalid("generated request body is required")
	}
	var request *T
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, invalid("generated request body does not match the operation schema")
	}
	if request == nil {
		return nil, invalid("generated request body must be an object")
	}
	return request, nil
}

func generatedRequestMap(body []byte) (map[string]any, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var request map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		return nil, invalid("invalid generated request body: %v", err)
	}
	if request == nil {
		return nil, invalid("generated request body must be an object")
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, invalid("invalid generated request body: %v", err)
	}
	return request, nil
}

func generatedConnectorBody(body []byte, requested string, priorResponse []byte) ([]byte, string, error) {
	var config map[string]json.RawMessage
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, "", invalid("invalid connector configuration: %v", err)
	}
	if config == nil {
		return nil, "", invalid("connector configuration must be an object")
	}
	canonical := ""
	var available struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if len(priorResponse) != 0 {
		if err := json.Unmarshal(priorResponse, &available); err != nil {
			return nil, "", invalid("invalid connector catalogue response: %v", err)
		}
		for candidate := range available.Data {
			if strings.EqualFold(candidate, requested) {
				canonical = candidate
				break
			}
		}
	}
	if canonical == "" {
		return nil, "", invalid("connector provider is absent from the live catalogue")
	}
	provider, err := json.Marshal(canonical)
	if err != nil {
		return nil, "", invalid("invalid connector provider: %v", err)
	}
	config["provider"] = provider
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, "", invalid("invalid connector configuration: %v", err)
	}
	return encoded, canonical, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func generatedPageSize(flags map[string][]string, cursor string) (*int64, error) {
	if cursor != "" {
		return nil, nil
	}
	value := first(flags["page-size"])
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, invalid("page-size must be an integer")
	}
	return &parsed, nil
}

func optionalBool(flags map[string][]string, name string) (*bool, error) {
	value := first(flags[name])
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, invalid("%s must be a boolean", name)
	}
	return &parsed, nil
}

func optionalTime(flags map[string][]string, name string) (*time.Time, error) {
	value := first(flags[name])
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, invalid("%s must be RFC3339", name)
	}
	return &parsed, nil
}

func requiredTime(arguments map[string][]string, name string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, first(arguments[name]))
	if err != nil {
		return time.Time{}, invalid("%s must be RFC3339", name)
	}
	return parsed, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
