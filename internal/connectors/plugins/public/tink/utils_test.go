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
			func(unscaledValueStr string, scale int, expectedValue *big.Int, expectedAsset string) {
				value, asset, err := MapTinkAmount(unscaledValueStr, strconv.Itoa(scale), "GBP")
				Expect(err).To(BeNil())
				Expect(value.Cmp(expectedValue)).To(Equal(0))
				Expect(*asset).To(Equal(expectedAsset))
			},
			Entry("negative value with scale 2", "-399", 2, big.NewInt(-399), "GBP/2"),
			Entry("negative value with scale 1", "-99", 1, big.NewInt(-99), "GBP/1"),
			Entry("negative value with scale 1", "-390", 1, big.NewInt(-390), "GBP/1"),
			Entry("positive value with scale 2", "389259", 2, big.NewInt(389259), "GBP/2"),
			Entry("positive value with scale 0", "3384", 0, big.NewInt(3384), "GBP/0"),
			Entry("positive value with scale 2", "332635", 2, big.NewInt(332635), "GBP/2"),
			Entry("positive value with negative scale -2", "132", -2, big.NewInt(13200), "GBP/0"),
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
	})
})
