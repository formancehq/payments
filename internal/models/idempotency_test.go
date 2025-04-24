package models_test

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestIdempotencyKey(t *testing.T) {
	t.Parallel()

	t.Run("generates consistent hash for same input", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
		}
		input := testStruct{
			Field1: "test",
			Field2: 123,
		}

		key1 := models.IdempotencyKey(input)
		key2 := models.IdempotencyKey(input)

		assert.Equal(t, key1, key2)
		assert.NotEmpty(t, key1)
	})

	t.Run("generates different hash for different inputs", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Field1 string `json:"field1"`
			Field2 int    `json:"field2"`
		}
		input1 := testStruct{
			Field1: "test1",
			Field2: 123,
		}
		input2 := testStruct{
			Field1: "test2",
			Field2: 456,
		}

		key1 := models.IdempotencyKey(input1)
		key2 := models.IdempotencyKey(input2)

		assert.NotEqual(t, key1, key2)
	})

	t.Run("generates expected hash for static input", func(t *testing.T) {
		t.Parallel()

		type staticStruct struct {
			ID string `json:"id"`
		}
		input := staticStruct{
			ID: "static-id-for-testing",
		}

		key := models.IdempotencyKey(input)

		expectedHash := "c0a0f5a0c3bcf2e0ce6f7d394a3adcee9f4c3ce0"
		assert.Equal(t, expectedHash, key)
	})
}
