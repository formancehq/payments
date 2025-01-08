package services

import (
	"fmt"
	"testing"

	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/stretchr/testify/require"
)

func TestHandleEngineErrors(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		typedError    bool
		expectedError error
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedError: nil,
		},
		{
			name:          "validation error",
			err:           fmt.Errorf("validation error: %w", engine.ErrValidation),
			expectedError: ErrValidation,
			typedError:    true,
		},
		{
			name:          "not found error",
			err:           fmt.Errorf("not found: %w", engine.ErrNotFound),
			expectedError: ErrNotFound,
			typedError:    true,
		},
		{
			name:          "other error",
			err:           fmt.Errorf("other error"),
			expectedError: fmt.Errorf("other error"),
			typedError:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := handleEngineErrors(test.err)
			if test.expectedError == nil {
				require.Nil(t, err)
			} else if test.typedError {
				require.ErrorIs(t, err, test.expectedError)
			} else {
				require.Equal(t, test.expectedError, err)
			}
		})
	}
}
