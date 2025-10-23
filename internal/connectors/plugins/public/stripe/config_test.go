package stripe

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
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

	Context("with valid configuration and explicit polling period", func() {
		BeforeEach(func() {
			// 45 minutes in nanoseconds
			payload = json.RawMessage(`{"apiKey":"sk_test_123","pollingPeriod":"45m"}`)
			expectedError = nil
		})

		It("should successfully unmarshal and validate with given polling period", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal("sk_test_123"))
			Expect(config.PollingPeriod).To(Equal(45 * time.Minute))
		})
	})

	Context("with missing polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"sk_test_123"}`)
			expectedError = nil
		})

		It("should default to 30 minutes", func() {
			Expect(err).To(BeNil())
			Expect(config.PollingPeriod).To(Equal(30 * time.Minute))
		})
	})

	Context("with polling period lower than minimum", func() {
		BeforeEach(func() {
			// 5 minutes in nanoseconds
			payload = json.RawMessage(`{"apiKey":"sk_test_123","pollingPeriod":"5m"}`)
			expectedError = nil
		})

		It("should coerce to minimum 20 minutes", func() {
			Expect(err).To(BeNil())
			Expect(config.PollingPeriod).To(Equal(20 * time.Minute))
		})
	})

	Context("with missing apiKey", func() {
		BeforeEach(func() {
			// 30 minutes in nanoseconds
			payload = json.RawMessage(`{"pollingPeriod":"30m"}`)
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

var _ = Describe("marshall", func() {
	It("keeps duration in string format", func() {
		config := &Config{
			APIKey:        "sk_test_123",
			PollingPeriod: 30 * time.Minute,
		}
		marshaledConfig, err := config.MarshalJSON()
		Expect(err).To(BeNil())
		Expect(string(marshaledConfig)).To(ContainSubstring(`"pollingPeriod":"30m0s"`))
		Expect(string(marshaledConfig)).To(ContainSubstring(`"apiKey":"sk_test_123"`))
	})
})
