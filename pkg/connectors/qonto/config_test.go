package qonto

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"

	"github.com/formancehq/payments/pkg/connector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("unmarshalAndValidateConfig", func() {
	var (
		payload       json.RawMessage
		expectedError error
		config        Config
		err           error

		defaultPollingPeriod connector.PollingPeriod
		longPollingPeriod    connector.PollingPeriod
	)

	BeforeEach(func() {
		var err error
		defaultPollingPeriod, err = connector.NewPollingPeriod("", connector.DefaultPollingPeriod, connector.MinimumPollingPeriod)
		Expect(err).To(BeNil())
		longPollingPeriod, err = connector.NewPollingPeriod("45m", connector.DefaultPollingPeriod, connector.MinimumPollingPeriod)
		Expect(err).To(BeNil())
	})

	JustBeforeEach(func() {
		config, err = unmarshalAndValidateConfig(payload)
	})

	Context("with valid configuration", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"clientID":"validClient","apiKey":"validApiKey","endpoint":"https://example.com","stagingToken":"token123"}`)
			expectedError = nil
		})

		It("should successfully unmarshal and validate", func() {
			Expect(err).To(BeNil())
			Expect(config.ClientID).To(Equal("validClient"))
			Expect(config.APIKey).To(Equal("validApiKey"))
			Expect(config.Endpoint).To(Equal("https://example.com"))
			Expect(config.StagingToken).To(Equal("token123"))
			Expect(config.PollingPeriod).To(Equal(defaultPollingPeriod))
		})
	})

	Context("with missing required fields", func() {
		When("clientID is missing", func() {
			BeforeEach(func() {
				payload = json.RawMessage(`{"apiKey":"validApiKey","endpoint":"https://example.com"}`)
			})

			It("should return a validation error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(validator.ValidationErrors{}))
			})
		})

		When("apiKey is missing", func() {
			BeforeEach(func() {
				payload = json.RawMessage(`{"clientID":"validClient","endpoint":"https://example.com"}`)
			})

			It("should return a validation error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(validator.ValidationErrors{}))
			})
		})

		When("endpoint is missing", func() {
			BeforeEach(func() {
				payload = json.RawMessage(`{"clientID":"validClient","apiKey":"validApiKey"}`)
			})

			It("should return a validation error", func() {
				Expect(err).To(HaveOccurred())
				Expect(err).To(BeAssignableToTypeOf(validator.ValidationErrors{}))
			})
		})
	})

	Context("with extra unknown fields", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"clientID":"validClient","apiKey":"validApiKey","endpoint":"https://example.com","unknownField":"value"}`)
			expectedError = nil
		})

		It("should ignore unknown fields and successfully unmarshal", func() {
			Expect(err).To(BeNil())
			Expect(config.ClientID).To(Equal("validClient"))
			Expect(config.APIKey).To(Equal("validApiKey"))
			Expect(config.Endpoint).To(Equal("https://example.com"))
			Expect(config.StagingToken).To(BeEmpty())
			Expect(config.PollingPeriod).To(Equal(defaultPollingPeriod))
		})
	})

	Context("with invalid JSON payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"clientID":invalidJson}`)
			expectedError = connector.ErrInvalidConfig
		})

		It("should return an unmarshalling error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(expectedError.Error())))
		})
	})

	Context("with empty payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(``)
			expectedError = connector.ErrInvalidConfig
		})

		It("should return an unmarshalling error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(expectedError.Error())))
		})
	})

	Context("with custom polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"clientID":"validClient","apiKey":"validApiKey","endpoint":"https://example.com","pollingPeriod":"45m"}`)
		})

		It("should parse and set the custom polling period", func() {
			Expect(err).To(BeNil())
			Expect(config.PollingPeriod).To(Equal(longPollingPeriod))
		})
	})

	Context("with invalid polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"clientID":"validClient","apiKey":"validApiKey","endpoint":"https://example.com","pollingPeriod":"not-a-duration"}`)
			expectedError = connector.ErrInvalidConfig
		})

		It("should return an error about invalid config", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(expectedError.Error())))
		})
	})
})
