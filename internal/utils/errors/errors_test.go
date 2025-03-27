package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	t.Parallel()

	var (
		causeError = errors.New("cause error")
		testError  = errors.New("test error")
	)
	type testCase struct {
		name        string
		err         error
		wantedCause error
		wantedError error
	}

	testCases := []testCase{
		{
			name:        "simple",
			err:         NewWrappedError(causeError, errors.New("test")),
			wantedCause: causeError,
			wantedError: causeError,
		},
		{
			name:        "double wrapped",
			err:         NewWrappedError(NewWrappedError(causeError, errors.New("test")), errors.New("test")),
			wantedCause: causeError,
			wantedError: causeError,
		},
		{
			name:        "double wrapped but other logical error should still work 1",
			err:         NewWrappedError(NewWrappedError(causeError, testError), errors.New("test")),
			wantedCause: causeError,
			wantedError: testError,
		},

		{
			name:        "double wrapped but other logical error should still work 1",
			err:         NewWrappedError(NewWrappedError(causeError, errors.New("test")), testError),
			wantedCause: causeError,
			wantedError: testError,
		},
		{
			name:        "standard error in wrapped error",
			err:         NewWrappedError(fmt.Errorf("%w", testError), errors.New("test")),
			wantedError: testError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.wantedError != nil {
				require.ErrorIs(t, tc.err, tc.wantedError)
			}

			if tc.wantedCause != nil {
				require.Equal(t, Cause(tc.err).Error(), tc.wantedCause.Error())
			}
		})
	}
}
