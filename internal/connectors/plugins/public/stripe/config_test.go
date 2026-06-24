package stripe

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("unmarshalAndValidateConfig", func() {
	var (
		payload       json.RawMessage
		expectedError error
		config        Config
		err           error
	)

	JustBeforeEach(func() {
		config, err = unmarshalAndValidateConfig(payload)
	})

	Context("with valid configuration", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"sk_test_123"}`)
			expectedError = nil
		})

		It("should successfully unmarshal and validate", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal("sk_test_123"))
		})
	})

	Context("with missing apiKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(BeAssignableToTypeOf(validator.ValidationErrors{}))
		})
	})

	Context("with invalid JSON payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":sk_test_123}`)
			expectedError = models.ErrInvalidConfig
		})

		It("should return an unmarshalling error wrapped as invalid config", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(expectedError.Error())))
		})
	})
})
