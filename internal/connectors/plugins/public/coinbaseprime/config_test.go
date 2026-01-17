package coinbaseprime

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
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id",
				"pollingPeriod": "45m"
			}`)
		})

		It("should successfully unmarshal and validate with given polling period", func() {
			Expect(err).To(BeNil())
			Expect(config.AccessKey).To(Equal("test-access-key"))
			Expect(config.Passphrase).To(Equal("test-passphrase"))
			Expect(config.SigningKey).To(Equal("test-signing-key"))
			Expect(config.PortfolioID).To(Equal("test-portfolio-id"))
			Expect(config.SvcAccountID).To(Equal("test-svc-account-id"))
			Expect(config.EntityID).To(Equal("test-entity-id"))
			Expect(config.PollingPeriod.Duration()).To(Equal(45 * time.Minute))
		})
	})

	Context("with missing polling period", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id"
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
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id",
				"pollingPeriod": "5m"
			}`)
		})

		It("should coerce to minimum 20 minutes", func() {
			Expect(err).To(BeNil())
			Expect(config.PollingPeriod.Duration()).To(Equal(20 * time.Minute))
		})
	})

	Context("with missing accessKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AccessKey"))
		})
	})

	Context("with missing passphrase", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"accessKey": "test-access-key",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Passphrase"))
		})
	})

	Context("with missing signingKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SigningKey"))
		})
	})

	Context("with missing portfolioId", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"svcAccountId": "test-svc-account-id",
				"entityId": "test-entity-id"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PortfolioID"))
		})
	})

	Context("with missing svcAccountId", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"entityId": "test-entity-id"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("SvcAccountID"))
		})
	})

	Context("with missing entityId", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"accessKey": "test-access-key",
				"passphrase": "test-passphrase",
				"signingKey": "test-signing-key",
				"portfolioId": "test-portfolio-id",
				"svcAccountId": "test-svc-account-id"
			}`)
		})

		It("should return a validation error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("EntityID"))
		})
	})

	Context("with invalid JSON payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"accessKey":invalid}`)
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
			"accessKey": "test-access-key",
			"passphrase": "test-passphrase",
			"signingKey": "test-signing-key",
			"portfolioId": "test-portfolio-id",
			"svcAccountId": "test-svc-account-id",
			"entityId": "test-entity-id",
			"pollingPeriod": "30m"
		}`)
		config, err := unmarshalAndValidateConfig(raw)
		Expect(err).To(BeNil())
		marshaledConfig, err := json.Marshal(config)
		Expect(err).To(BeNil())
		Expect(string(marshaledConfig)).To(ContainSubstring(`"pollingPeriod":"30m0s"`))
		Expect(string(marshaledConfig)).To(ContainSubstring(`"accessKey":"test-access-key"`))
	})
})
