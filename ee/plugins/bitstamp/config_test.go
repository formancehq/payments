package bitstamp

import (
	"encoding/json"

	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("unmarshalAndValidateConfig", func() {
	const (
		apiKeyPlaceholder    = "bitstamp-api-key-placeholder"
		apiSecretPlaceholder = "bitstamp-api-secret-placeholder"
	)

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
			payload = json.RawMessage(`{
				"apiKey": "bitstamp-api-key-placeholder",
				"apiSecret": "bitstamp-api-secret-placeholder",
				"endpoint": "https://www.bitstamp.net"
			}`)
		})

		It("populates the struct", func() {
			Expect(err).To(BeNil())
			Expect(config.APIKey).To(Equal(apiKeyPlaceholder))
			Expect(config.APISecret).To(Equal(apiSecretPlaceholder))
			Expect(config.Endpoint).To(Equal("https://www.bitstamp.net"))
		})
	})

	Context("with missing apiKey", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiSecret": "bitstamp-api-secret-placeholder"}`)
		})

		It("returns a validation error naming APIKey", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
	})

	Context("with missing apiSecret", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey": "bitstamp-api-key-placeholder"}`)
		})

		It("returns a validation error naming APISecret", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APISecret"))
		})
	})

	Context("with invalid endpoint URL", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"apiKey": "bitstamp-api-key-placeholder",
				"apiSecret": "bitstamp-api-secret-placeholder",
				"endpoint": "not a url"
			}`)
		})

		It("returns a validation error on Endpoint", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Endpoint"))
		})
	})

	Context("with empty endpoint (defaults to production)", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{
				"apiKey": "bitstamp-api-key-placeholder",
				"apiSecret": "bitstamp-api-secret-placeholder",
				"endpoint": ""
			}`)
		})

		It("accepts an empty endpoint (omitempty)", func() {
			Expect(err).To(BeNil())
			Expect(config.Endpoint).To(Equal(""))
		})
	})

	Context("with invalid JSON payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{"apiKey": invalid}`)
		})

		It("wraps as an invalid-config error", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(models.ErrInvalidConfig.Error()))
		})
	})

	Context("with empty payload", func() {
		BeforeEach(func() {
			payload = json.RawMessage(`{}`)
		})

		It("flags every required field", func() {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("APIKey"))
		})
	})
})
