package moneycorp

import (
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Moneycorp nil response conversion", func() {
	Context("transferToPayment", func() {
		It("returns an invalid-request error for a nil transfer", func() {
			payment, err := transferToPayment(nil)
			Expect(payment).To(BeNil())
			Expect(err).To(MatchError(models.ErrInvalidRequest))
		})
	})

	Context("payoutToPayment", func() {
		It("returns an invalid-request error for a nil payout", func() {
			payment, err := payoutToPayment(nil)
			Expect(payment).To(BeNil())
			Expect(err).To(MatchError(models.ErrInvalidRequest))
		})
	})
})
