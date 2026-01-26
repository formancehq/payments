package kraken

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

	Context("with valid configuration and explicit polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"endpoint": "https://api.kraken.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"pollingPeriod": "45m"
			}`)
		})

		It("should successfully unmarshal and validate with given polling period", func() {
			Expect(err).To(BeNil())
			Expect(config.Endpoint).To(Equal("https://api.kraken.com"))
			Expect(config.PublicKey).To(Equal("test-public-key"))
			Expect(config.PrivateKey).To(Equal("test-private-key"))
			Expect(config.PollingPeriod.Duration()).To(Equal(45 * time.Minute))
		})
	})

	Context("with missing polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"endpoint": "https://api.kraken.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key"
			}`)
		})

		It("should default to 30 minutes", func() {
			Expect(err).To(BeNil())
			Expect(config.PollingPeriod.Duration()).To(Equal(30 * time.Minute))
		})
	})

	Context("with polling period lower than minimum", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"endpoint": "https://api.kraken.com",
				"publicKey": "test-public-key",
				"privateKey": "test-private-key",
				"pollingPeriod": "5m"
			}`)
		})

		It("should coerce to minimum 20 minutes", func() {
			Expect(err).To(BeNil())
			Expect(config.PollingPeriod.Duration()).To(Equal(20 * time.Minute))
		})
	})

	Context("with missing endpoint", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"publicKey": "test-public-key",
				"privateKey": "test-private-key"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})
	})

	Context("with missing publicKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"endpoint": "https://api.kraken.com",
				"privateKey": "test-private-key"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PublicKey"))
		})
	})

	Context("with missing privateKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"endpoint": "https://api.kraken.com",
				"publicKey": "test-public-key"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PrivateKey"))
		})
	})

	Context("with invalid JSON payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"endpoint":invalid}`)
		})

		It("should return an unmarshalling error wrapped as invalid config", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(models.ErrInvalidConfig.Error()))
		})
	})
})

var _ = Describe("marshall", func() {
	It("keeps duration in string format", func() {
		raw := json.RawMessage(`{
			"endpoint": "https://api.kraken.com",
			"publicKey": "test-public-key",
			"privateKey": "test-private-key",
			"pollingPeriod": "30m"
		}`)
		config, err := unmarshalAndValidateConfig(raw)
		Expect(err).To(BeNil())
		marshaledConfig, err := json.Marshal(config)
		Expect(err).To(BeNil())
		Expect(string(marshaledConfig)).To(ContainSubstring(`"pollingPeriod":"30m0s"`))
		Expect(string(marshaledConfig)).To(ContainSubstring(`"endpoint":"https://api.kraken.com"`))
	})
})
