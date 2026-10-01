// Package producthttp adapts generated HTTP clients to one host-owned product
// operation. It never resolves an endpoint, injects credentials, or retries.
package producthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const sentinelHost = "product.invalid"

// Client is an immutable binding between one generated HTTP client operation
// and the host request boundary.
type Client struct {
	host       sdk.Host
	policy     sdk.OperationPolicy
	capability string
}

// New validates and freezes one generated HTTP operation binding.
func New(host sdk.Host, policy sdk.OperationPolicy, capability string) (*Client, error) {
	if host == nil || capability == sdk.HostCapabilityGeneratedClientV1 || sdk.ValidateGeneratedClientOperationPolicy(policy) != nil || policy.HTTP == nil {
		return nil, failure(sdk.FailureDescriptorInvalid, "invalid generated HTTP operation binding")
	}
	policy = clonePolicy(policy)
	return &Client{host: host, policy: policy, capability: capability}, nil
}

// Do maps one generated HTTP request to exactly one Host.Request.
func (c *Client) Do(request *http.Request) (*http.Response, error) {
	logical, contentType, err := c.normalizeRequest(request)
	if err != nil {
		return nil, err
	}

	generated := c.policy.HTTP.GeneratedClient
	body, err := readRequestBody(request, generated.MaxRequestBytes)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 && contentType == "" {
		return nil, failure(sdk.FailureInvalidArgument, "request body requires a declared content type")
	}
	logical.Body = body
	logical.ContentType = contentType

	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	responses, err := c.host.Request(request.Context(), sdk.Request{
		Service: c.policy.Service, Capability: c.capability, Operation: c.policy.ID, HTTP: logical,
	})
	if err != nil {
		return nil, mapHostError(err)
	}
	return c.readResponse(request.Context(), responses)
}

func (c *Client) normalizeRequest(request *http.Request) (*sdk.HTTPRequest, string, error) {
	if request == nil || request.URL == nil || request.Method != c.policy.HTTP.Method || !validSentinelURL(request) || !validRequestFields(request) {
		return nil, "", failure(sdk.FailureInvalidArgument, "invalid generated HTTP request")
	}
	generated := c.policy.HTTP.GeneratedClient
	if request.ContentLength < -1 || request.ContentLength > generated.MaxRequestBytes {
		return nil, "", failure(sdk.FailureInvalidArgument, "invalid request content length")
	}

	path, err := sdk.NormalizeGeneratedHTTPPath(request.URL.Path, request.URL.RawPath, *c.policy.HTTP)
	if err != nil {
		return nil, "", failure(sdk.FailureInvalidArgument, "invalid request path")
	}
	query, err := sdk.NormalizeGeneratedHTTPQuery(request.URL.RawQuery)
	if err != nil {
		return nil, "", failure(sdk.FailureInvalidArgument, "invalid request query")
	}
	headerInput, contentType, err := splitContentType(request.Header)
	if err != nil {
		return nil, "", failure(sdk.FailureInvalidArgument, "invalid request content type")
	}
	headers, err := sdk.NormalizeGeneratedHTTPHeaders(headerInput, *c.policy.HTTP)
	if err != nil {
		return nil, "", failure(sdk.FailureInvalidArgument, "invalid request headers")
	}
	if contentType != "" {
		contentType, err = sdk.NormalizeGeneratedHTTPContentType(contentType)
		if err != nil || !declaresContentType(generated.RequestContentTypes, contentType) {
			return nil, "", failure(sdk.FailureInvalidArgument, "invalid request content type")
		}
	}
	if request.ContentLength > 0 && contentType == "" {
		return nil, "", failure(sdk.FailureInvalidArgument, "request body requires a declared content type")
	}
	return &sdk.HTTPRequest{Method: request.Method, Path: path, Query: query, Headers: headers}, contentType, nil
}

func validRequestFields(request *http.Request) bool {
	emptyTuple := request.Host == "" && request.Proto == "" && request.ProtoMajor == 0 && request.ProtoMinor == 0
	constructorTuple := request.Host == sentinelHost && request.Proto == "HTTP/1.1" && request.ProtoMajor == 1 && request.ProtoMinor == 1
	return (emptyTuple || constructorTuple) && request.RequestURI == "" &&
		request.RemoteAddr == "" && request.Pattern == "" && len(request.TransferEncoding) == 0 && request.Trailer == nil && request.TLS == nil &&
		request.Response == nil && request.Form == nil && request.PostForm == nil && request.MultipartForm == nil && request.Cancel == nil && !request.Close
}

func validSentinelURL(request *http.Request) bool {
	url := request.URL
	return asciiEqualFold(url.Scheme, "https") && asciiEqualFold(url.Host, sentinelHost) && asciiEqualFold(url.Hostname(), sentinelHost) &&
		url.Port() == "" && url.User == nil && url.Opaque == "" && url.Fragment == "" && url.RawFragment == "" && !url.ForceQuery && !url.OmitHost
}

func asciiEqualFold(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range len(left) {
		leftByte, rightByte := left[index], right[index]
		if leftByte >= 'A' && leftByte <= 'Z' {
			leftByte += 'a' - 'A'
		}
		if rightByte >= 'A' && rightByte <= 'Z' {
			rightByte += 'a' - 'A'
		}
		if leftByte != rightByte {
			return false
		}
	}
	return true
}

func splitContentType(input http.Header) (map[string][]string, string, error) {
	headers := make(map[string][]string, len(input))
	contentType := ""
	foundContentType := false
	for name, values := range input {
		if strings.EqualFold(name, "content-type") {
			if foundContentType || len(values) != 1 {
				return nil, "", errors.New("content type must be singular")
			}
			foundContentType = true
			contentType = values[0]
			continue
		}
		headers[name] = append([]string(nil), values...)
	}
	return headers, contentType, nil
}

func declaresContentType(declared []string, actual string) bool {
	for _, candidate := range declared {
		canonical, err := sdk.NormalizeGeneratedHTTPContentType(candidate)
		if err == nil && canonical == actual {
			return true
		}
	}
	return false
}

func readRequestBody(request *http.Request, maxBytes int64) ([]byte, error) {
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	if request.Body == nil || request.Body == http.NoBody {
		if request.ContentLength > 0 {
			return nil, failure(sdk.FailureInvalidArgument, "request content length does not match body")
		}
		return []byte{}, nil
	}

	reader := io.LimitReader(request.Body, maxBytes+1)
	body, readErr := io.ReadAll(reader)
	closeErr := request.Body.Close()
	if err := request.Context().Err(); err != nil {
		return nil, err
	}
	if readErr != nil {
		return nil, failure(sdk.FailureInvalidArgument, "could not read request body")
	}
	if int64(len(body)) > maxBytes {
		return nil, failure(sdk.FailureInputTooLarge, "request body is too large")
	}
	if request.ContentLength >= 0 && request.ContentLength != int64(len(body)) {
		return nil, failure(sdk.FailureInvalidArgument, "request content length does not match body")
	}
	if closeErr != nil {
		return nil, failure(sdk.FailureInvalidArgument, "could not close request body")
	}
	return append([]byte(nil), body...), nil
}

func (c *Client) readResponse(ctx context.Context, responses sdk.Responses) (*http.Response, error) {
	if responses == nil {
		return nil, failure(sdk.FailureProductResponseFailed, "product response failed")
	}
	response, err := responses.Recv()
	if err != nil {
		if contextError(err) {
			return nil, err
		}
		return nil, failure(sdk.FailureProductResponseFailed, "product response failed")
	}
	terminalResponse, terminalErr := responses.Recv()
	if terminalErr != io.EOF || responsePresent(terminalResponse) {
		if contextError(terminalErr) {
			return nil, terminalErr
		}
		return nil, failure(sdk.FailureProductResponseFailed, "product response failed")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limits := c.policy.HTTP.GeneratedClient.ResponseLimits
	if sdk.ResponseStreamMetadataOf(responses) != (sdk.ResponseStreamMetadata{}) || response.Status < 100 || response.Status > 599 {
		return nil, failure(sdk.FailureProductResponseFailed, "product response failed")
	}
	if response.Status < 200 || response.Status > 299 {
		return nil, productHTTPFailure(response.Status)
	}
	if int64(len(response.Body)) > limits.MaxMessageBytes || int64(len(response.Body)) > limits.MaxAggregateBytes {
		return nil, failure(sdk.FailureProductResponseFailed, "product response failed")
	}

	contentType := ""
	if response.ContentType != "" {
		var err error
		contentType, err = sdk.NormalizeGeneratedHTTPContentType(response.ContentType)
		if err != nil {
			return nil, failure(sdk.FailureProductResponseFailed, "product response failed")
		}
	}
	body := append([]byte(nil), response.Body...)
	statusCode := int(response.Status)
	status := fmt.Sprintf("%d", statusCode)
	if text := http.StatusText(statusCode); text != "" {
		status += " " + text
	}
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		Status: status, StatusCode: statusCode, Header: header, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)),
	}, nil
}

func productHTTPFailure(status int32) sdk.Failure {
	details, _ := json.Marshal(struct {
		HTTPStatus int32 `json:"httpStatus"`
		Details    any   `json:"details"`
	}{HTTPStatus: status})
	return sdk.Failure{
		Code:      string(sdk.FailureProductHTTPError),
		Message:   fmt.Sprintf("product request returned HTTP %d", status),
		Details:   details,
		Retryable: status >= http.StatusInternalServerError,
	}
}

func responsePresent(response sdk.Response) bool {
	return response.Body != nil || response.Status != 0 || response.ContentType != ""
}

func mapHostError(err error) error {
	if contextError(err) {
		return err
	}
	var value sdk.Failure
	if errors.As(err, &value) && sdk.FailureCode(value.Code).Valid() {
		return value
	}
	var pointer *sdk.Failure
	if errors.As(err, &pointer) && pointer != nil && sdk.FailureCode(pointer.Code).Valid() {
		return pointer
	}
	return failure(sdk.FailureHostRequestFailed, "host request failed")
}

func contextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func failure(code sdk.FailureCode, message string) sdk.Failure {
	return sdk.Failure{Code: string(code), Message: message, Retryable: false}
}

func clonePolicy(policy sdk.OperationPolicy) sdk.OperationPolicy {
	clone := policy
	clone.Scopes = append([]string(nil), policy.Scopes...)
	if policy.HTTP != nil {
		httpPolicy := *policy.HTTP
		if policy.HTTP.GeneratedClient != nil {
			generated := *policy.HTTP.GeneratedClient
			generated.RequestContentTypes = append([]string(nil), policy.HTTP.GeneratedClient.RequestContentTypes...)
			generated.RequestHeaders = append([]string(nil), policy.HTTP.GeneratedClient.RequestHeaders...)
			httpPolicy.GeneratedClient = &generated
		}
		clone.HTTP = &httpPolicy
	}
	return clone
}
