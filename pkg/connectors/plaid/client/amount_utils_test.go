package client_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/formancehq/payments/pkg/connectors/plaid/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAmountUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Amount Utils Suite")
}

var _ = Describe("Amount Utils", func() {
	Describe("TranslatePlaidAmount", func() {
		Context("with valid currency codes", func() {
			It("should convert USD amounts correctly", func() {
				amount := 123.45
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(12345))) // 123.45 * 100 (2 decimal places)
				Expect(assetName).To(Equal("USD/2"))
			})

			It("should convert JPY amounts correctly (no decimal places)", func() {
				amount := 1000.0
				currencyCode := "JPY"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(1000))) // 1000 * 1 (0 decimal places)
				Expect(assetName).To(Equal("JPY/0"))
			})

			It("should convert BHD amounts correctly (3 decimal places)", func() {
				amount := 1.234
				currencyCode := "BHD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(1234))) // 1.234 * 1000 (3 decimal places)
				Expect(assetName).To(Equal("BHD/3"))
			})

			It("should handle zero amounts", func() {
				amount := 0.0
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(0)))
				Expect(assetName).To(Equal("USD/2"))
			})

			It("should handle negative amounts", func() {
				amount := -123.45
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(-12345))) // -123.45 * 100
				Expect(assetName).To(Equal("USD/2"))
			})

			It("should handle very small amounts", func() {
				amount := 0.01
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(1))) // 0.01 * 100
				Expect(assetName).To(Equal("USD/2"))
			})

			It("should handle large amounts", func() {
				amount := 999999.99
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(99999999))) // 999999.99 * 100
				Expect(assetName).To(Equal("USD/2"))
			})

			It("should handle amounts with many decimal places", func() {
				amount := 123.456789
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				// This should FAIL because 123.456789 has more than 2 decimal places for USD
				Expect(err).ToNot(BeNil())
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})

			It("should handle EUR amounts correctly", func() {
				amount := 50.75
				currencyCode := "EUR"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(5075))) // 50.75 * 100
				Expect(assetName).To(Equal("EUR/2"))
			})

			It("should handle GBP amounts correctly", func() {
				amount := 25.50
				currencyCode := "GBP"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(2550))) // 25.50 * 100
				Expect(assetName).To(Equal("GBP/2"))
			})
		})

		Context("with invalid currency codes", func() {
			It("should return error for unsupported currency", func() {
				amount := 100.0
				currencyCode := "INVALID"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).ToNot(BeNil())
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})

			It("should return error for empty currency code", func() {
				amount := 100.0
				currencyCode := ""

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).ToNot(BeNil())
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})
		})

		Context("with special float values", func() {
			It("should handle NaN values", func() {
				amount := math.NaN()
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).ToNot(BeNil())
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})

			It("should handle positive infinity", func() {
				amount := math.Inf(1)
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).ToNot(BeNil())
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})

			It("should handle negative infinity", func() {
				amount := math.Inf(-1)
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).ToNot(BeNil())
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})
		})

		Context("with various precision currencies", func() {
			It("should handle KWD (3 decimal places)", func() {
				amount := 1.234
				currencyCode := "KWD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(1234))) // 1.234 * 1000
				Expect(assetName).To(Equal("KWD/3"))
			})

			It("should handle OMR (3 decimal places)", func() {
				amount := 5.678
				currencyCode := "OMR"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(5678))) // 5.678 * 1000
				Expect(assetName).To(Equal("OMR/3"))
			})

			It("should handle JOD (3 decimal places)", func() {
				amount := 10.123
				currencyCode := "JOD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).To(BeNil())
				Expect(resultAmount).To(Equal(big.NewInt(10123))) // 10.123 * 1000
				Expect(assetName).To(Equal("JOD/3"))
			})
		})

		Context("with boundary values", func() {
			It("should handle maximum float64 value", func() {
				amount := 1.7976931348623157e+308
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				// This should SUCCEED with a very large number (max float64 is valid)
				Expect(err).To(BeNil())
				Expect(resultAmount).ToNot(BeNil())
				Expect(assetName).To(Equal("USD/2"))
				// The result should be a very large number representing the max float64 in cents
				Expect(resultAmount.Cmp(big.NewInt(0))).To(Equal(1)) // Should be positive
			})

			It("should handle minimum positive float64 value", func() {
				amount := 4.9406564584124654e-324
				currencyCode := "USD"

				resultAmount, assetName, err := client.TranslatePlaidAmount(amount, currencyCode)

				Expect(err).ToNot(BeNil())
				Expect(err).To(MatchError("invalid precision"))
				Expect(resultAmount).To(BeNil())
				Expect(assetName).To(Equal(""))
			})
		})
	})
})
