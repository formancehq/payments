package coinbaseprime

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("unmarshalAndValidateConfig", func() {
	const apiKeyPlaceholder = "coinbase-api-key-placeholder"

	var (
		payload json.RawMessage
		config  Config
		err     error
	)

	JustBeforeEach(func() {
		config, err = unmarshalAndValidateConfig(payload)
	})

	Context("with valid configuration", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"coinbase-api-key-placeholder","apiSecret":"dGVzdC1zZWNyZXQ=","passphrase":"test-pass","portfolioId":"portfolio-123","pollingPeriod":"45m"}`)
		})

		It("should successfully unmarshal and validate", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal(apiKeyPlaceholder))
			Expect(config.APISecret).To(Equal("dGVzdC1zZWNyZXQ="))
			Expect(config.Passphrase).To(Equal("test-pass"))
			Expect(config.PortfolioID).To(Equal("portfolio-123"))
			Expect(config.PollingPeriod.Duration()).To(Equal(45 * time.Minute))
		})
	})

	Context("with valid configuration and default polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"coinbase-api-key-placeholder","apiSecret":"dGVzdC1zZWNyZXQ=","passphrase":"test-pass","portfolioId":"portfolio-123"}`)
		})

		It("should use default polling period", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal(apiKeyPlaceholder))
			Expect(config.PortfolioID).To(Equal("portfolio-123"))
			// Default polling period should be applied
			Expect(config.PollingPeriod.Duration()).ToNot(Equal(0))
		})
	})

	Context("with missing apiKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiSecret":"dGVzdA==","passphrase":"test","portfolioId":"portfolio-123"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
	})

	Context("with missing apiSecret", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test","passphrase":"test","portfolioId":"portfolio-123"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})
	})

	Context("with missing passphrase", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test","apiSecret":"dGVzdA==","portfolioId":"portfolio-123"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Passphrase"))
		})
	})

	Context("with missing portfolioId", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test","apiSecret":"dGVzdA==","passphrase":"test"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PortfolioID"))
		})
	})

	Context("with non-standard base64 apiSecret (Coinbase Prime format)", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test","apiSecret":"/NqFL5Fk4qM=oWgWLK5tib6H0RCUcndurw==","passphrase":"test","portfolioId":"portfolio-123"}`)
		})

		It("should accept secrets with padding in the middle", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("with invalid JSON payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":test}`)
		})

		It("should return an unmarshalling error wrapped as invalid config", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(models.ErrInvalidConfig.Error()))
		})
	})

	Context("with empty payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{}`)
		})

		It("should return validation errors for all required fields", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
	})
})
