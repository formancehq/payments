package models

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("missing name", func(t *testing.T) {
		config := Config{}
		err := config.Validate()
		require.Error(t, err)
		require.Equal(t, errors.New("name is required"), err)
	})

	t.Run("invalid polling period", func(t *testing.T) {
		config := Config{
			Name:          "test",
			PollingPeriod: 2 * time.Second,
		}
		err := config.Validate()
		require.Error(t, err)
		require.Equal(t, errors.New("polling period must be at least 30 seconds"), err)
	})

	t.Run("valid config", func(t *testing.T) {
		config := Config{
			Name:          "test",
			PollingPeriod: 30 * time.Second,
		}
		err := config.Validate()
		require.NoError(t, err)
	})
}
