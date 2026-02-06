package coinbase

import (
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("unmarshalAndValidateConfig", func() {
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
			payload = json.RawMessage(`{"apiKey":"test-key","apiSecret":"dGVzdC1zZWNyZXQ=","passphrase":"test-pass","pollingPeriod":"45m"}`)
		})

		It("should successfully unmarshal and validate", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal("test-key"))
			Expect(config.APISecret).To(Equal("dGVzdC1zZWNyZXQ="))
			Expect(config.Passphrase).To(Equal("test-pass"))
			Expect(config.PollingPeriod.Duration()).To(Equal(45 * time.Minute))
		})
	})

	Context("with valid configuration and default polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test-key","apiSecret":"dGVzdC1zZWNyZXQ=","passphrase":"test-pass"}`)
		})

		It("should use default polling period", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal("test-key"))
			// Default polling period should be applied
			Expect(config.PollingPeriod.Duration()).ToNot(Equal(0))
		})
	})

	Context("with missing apiKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiSecret":"test","passphrase":"test"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
	})

	Context("with missing apiSecret", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test","passphrase":"test"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})
	})

	Context("with missing passphrase", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey":"test","apiSecret":"test"}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Passphrase"))
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
