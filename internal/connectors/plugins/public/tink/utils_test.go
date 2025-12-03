package tink

import (
	"math/big"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MapTinkAmount", func() {
	Context("with GBP currency", func() {
		DescribeTable("should map Tink amounts correctly",
			func(unscaledValueStr string, scale int, currency string, expectedValue *big.Int, expectedAsset string) {
				value, asset, err := MapTinkAmount(unscaledValueStr, strconv.Itoa(scale), currency)
				Expect(err).To(BeNil())
				Expect(value).To(Equal(expectedValue))
				Expect(*asset).To(Equal(expectedAsset))
			},
			Entry("negative value with scale 2", "-399", 2, "GBP", big.NewInt(-399), "GBP/2"),
			Entry("negative value with scale 1", "-99", 1, "GBP", big.NewInt(-990), "GBP/2"),
			Entry("negative value with scale 1", "-390", 1, "GBP", big.NewInt(-3900), "GBP/2"),
			Entry("positive value with scale 2", "389259", 2, "GBP", big.NewInt(389259), "GBP/2"),
			Entry("positive value with scale 4", "3000", 4, "GBP", big.NewInt(30), "GBP/2"),

			Entry("positive value with scale 0", "3384", 0, "GBP", big.NewInt(338400), "GBP/2"),
			Entry("positive value with scale 2", "332635", 2, "GBP", big.NewInt(332635), "GBP/2"),
			Entry("positive value with negative scale -2", "132", -2, "GBP", big.NewInt(1_320_000), "GBP/2"),

			Entry("currency with precision of 3", "132", 2, "JOD", big.NewInt(1320), "JOD/3"),
		)
	})

	Context("error cases", func() {
		It("should return error for invalid unscaled value", func() {
			value, asset, err := MapTinkAmount("not-a-number", "2", "GBP")
			Expect(err).ToNot(BeNil())
			Expect(value).To(BeNil())
			Expect(asset).To(BeNil())
		})

		It("should return error for invalid scale", func() {
			value, asset, err := MapTinkAmount("100", "not-a-number", "GBP")
			Expect(err).ToNot(BeNil())
			Expect(value).To(BeNil())
			Expect(asset).To(BeNil())
		})

		It("should return error for invalid currency", func() {
			value, asset, err := MapTinkAmount("100", "2", "INVALID")
			Expect(err).ToNot(BeNil())
			Expect(value).To(BeNil())
			Expect(asset).To(BeNil())
		})
	})
})
