package api_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	v2 "github.com/formancehq/payments/internal/api/v2"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Every error code a handler can emit must be declared in the OpenAPI enum the
// corresponding error response points at. When it is not, generated clients fail
// to deserialize the response ("invalid value for <Enum>: <CODE>") and callers
// that treat an unmarshal failure as retryable retry forever instead of
// surfacing the error.
func TestErrorCodesAreDeclaredInOpenAPIEnums(t *testing.T) {
	t.Parallel()

	spec := loadOpenAPISpec(t)

	for _, tc := range []struct {
		enum  string
		codes []string
	}{
		{
			// referenced by PaymentsErrorResponse, used by every v1/v2 operation
			enum: "PaymentsErrorsEnum",
			codes: []string{
				api.ErrorInternal,
				v2.ErrValidation,
				v2.ErrInvalidID,
				v2.ErrMissingOrInvalidBody,
				v2.ErrUniqueReference,
				v2.ErrConnectorCapabilityNotSupported,
				api.ErrorCodeNotFound,
			},
		},
		{
			// referenced by V3ErrorResponse, used by every v3 operation
			enum: "V3ErrorsEnum",
			codes: []string{
				api.ErrorInternal,
				v3.ErrValidation,
				v3.ErrInvalidID,
				v3.ErrMissingOrInvalidBody,
				v3.ErrUniqueReference,
				v3.ErrConnectorCapabilityNotSupported,
				api.ErrorCodeNotFound,
			},
		},
	} {
		t.Run(tc.enum, func(t *testing.T) {
			t.Parallel()

			declared := declaredEnumValues(t, spec, tc.enum)
			for _, code := range tc.codes {
				require.Containsf(t, declared, code,
					"%s is emitted by a handler but not declared in %s; "+
						"add it to the OpenAPI source and run `just openapi generate-sdk`",
					code, tc.enum)
			}
		})
	}
}

func loadOpenAPISpec(t *testing.T) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "openapi.yaml"))
	require.NoError(t, err)

	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &spec))

	return spec
}

func declaredEnumValues(t *testing.T, spec map[string]any, name string) []string {
	t.Helper()

	components, ok := spec["components"].(map[string]any)
	require.True(t, ok, "components missing from openapi.yaml")

	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok, "components.schemas missing from openapi.yaml")

	schema, ok := schemas[name].(map[string]any)
	require.Truef(t, ok, "schema %s missing from openapi.yaml", name)

	raw, ok := schema["enum"].([]any)
	require.Truef(t, ok, "schema %s has no enum", name)

	values := make([]string, 0, len(raw))
	for _, v := range raw {
		value, ok := v.(string)
		require.Truef(t, ok, "schema %s has a non-string enum value %v", name, v)
		values = append(values, value)
	}

	return values
}
