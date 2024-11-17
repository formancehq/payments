package models

type PaymentInitiationRelatedPayments struct {
	// Payment Initiation ID
	PaymentInitiationID PaymentInitiationID `json:"paymentInitiationID"`

	// Related Payment ID
	PaymentID PaymentID `json:"paymentID"`
}
