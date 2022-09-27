package dummypay

import payments "github.com/numary/payments/pkg"

// payment represents a payment structure used in the generated files.
type payment struct {
	payments.Data
	Reference string `json:"reference" bson:"reference"`
	Type      string `json:"type" bson:"type"`
}
