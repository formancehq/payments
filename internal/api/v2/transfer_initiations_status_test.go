package v2

import (
	"github.com/formancehq/payments/pkg/domain/models"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// v2's TransferInitiationStatus enum is frozen and narrower than v3's, so every
// status v3 added afterwards has to be reported as its closest v2 equivalent.
// NOT_INITIATED refines FAILED, and v2 clients drive their retry logic off
// FAILED, so that is what they must keep seeing.
var _ = Describe("API v2 Transfer Initiation Status Translation", func() {
	Context("translateLastStatus", func() {
		It("reports NOT_INITIATED as FAILED", func(ctx SpecContext) {
			Expect(translateLastStatus(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED)).To(Equal("FAILED"))
		})

		It("reports SCHEDULED_FOR_PROCESSING as PROCESSING", func(ctx SpecContext) {
			Expect(translateLastStatus(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING)).To(Equal("PROCESSING"))
		})

		It("passes through the statuses v2 already knows", func(ctx SpecContext) {
			for status, expected := range map[models.PaymentInitiationAdjustmentStatus]string{
				models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_WAITING_FOR_VALIDATION: "WAITING_FOR_VALIDATION",
				models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING:             "PROCESSING",
				models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED:              "PROCESSED",
				models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED:                 "FAILED",
				models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REJECTED:               "REJECTED",
				models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED:               "REVERSED",
			} {
				Expect(translateLastStatus(status)).To(Equal(expected))
			}
		})
	})

	Context("translateStatus", func() {
		It("keeps NOT_INITIATED in the adjustment list, as FAILED", func(ctx SpecContext) {
			// Unlike SCHEDULED_FOR_PROCESSING, this one carries a failure:
			// dropping it would hide the rejection from v2 clients entirely.
			status, toSend := translateStatus(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_NOT_INITIATED)
			Expect(toSend).To(BeTrue())
			Expect(status).To(Equal("FAILED"))
		})

		It("drops SCHEDULED_FOR_PROCESSING", func(ctx SpecContext) {
			_, toSend := translateStatus(models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_SCHEDULED_FOR_PROCESSING)
			Expect(toSend).To(BeFalse())
		})
	})
})
