// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"encoding/json"
	"fmt"
)

type V3PaymentStatusEnum string

const (
	V3PaymentStatusEnumUnknown           V3PaymentStatusEnum = "UNKNOWN"
	V3PaymentStatusEnumPending           V3PaymentStatusEnum = "PENDING"
	V3PaymentStatusEnumSucceeded         V3PaymentStatusEnum = "SUCCEEDED"
	V3PaymentStatusEnumCancelled         V3PaymentStatusEnum = "CANCELLED"
	V3PaymentStatusEnumFailed            V3PaymentStatusEnum = "FAILED"
	V3PaymentStatusEnumExpired           V3PaymentStatusEnum = "EXPIRED"
	V3PaymentStatusEnumRefunded          V3PaymentStatusEnum = "REFUNDED"
	V3PaymentStatusEnumRefundedFailure   V3PaymentStatusEnum = "REFUNDED_FAILURE"
	V3PaymentStatusEnumRefundReversed    V3PaymentStatusEnum = "REFUND_REVERSED"
	V3PaymentStatusEnumDispute           V3PaymentStatusEnum = "DISPUTE"
	V3PaymentStatusEnumDisputeWon        V3PaymentStatusEnum = "DISPUTE_WON"
	V3PaymentStatusEnumDisputeLost       V3PaymentStatusEnum = "DISPUTE_LOST"
	V3PaymentStatusEnumAmountAdjustement V3PaymentStatusEnum = "AMOUNT_ADJUSTEMENT"
	V3PaymentStatusEnumAuthorisation     V3PaymentStatusEnum = "AUTHORISATION"
	V3PaymentStatusEnumCapture           V3PaymentStatusEnum = "CAPTURE"
	V3PaymentStatusEnumCaptureFailed     V3PaymentStatusEnum = "CAPTURE_FAILED"
	V3PaymentStatusEnumOther             V3PaymentStatusEnum = "OTHER"
)

func (e V3PaymentStatusEnum) ToPointer() *V3PaymentStatusEnum {
	return &e
}
func (e *V3PaymentStatusEnum) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch v {
	case "UNKNOWN":
		fallthrough
	case "PENDING":
		fallthrough
	case "SUCCEEDED":
		fallthrough
	case "CANCELLED":
		fallthrough
	case "FAILED":
		fallthrough
	case "EXPIRED":
		fallthrough
	case "REFUNDED":
		fallthrough
	case "REFUNDED_FAILURE":
		fallthrough
	case "REFUND_REVERSED":
		fallthrough
	case "DISPUTE":
		fallthrough
	case "DISPUTE_WON":
		fallthrough
	case "DISPUTE_LOST":
		fallthrough
	case "AMOUNT_ADJUSTEMENT":
		fallthrough
	case "AUTHORISATION":
		fallthrough
	case "CAPTURE":
		fallthrough
	case "CAPTURE_FAILED":
		fallthrough
	case "OTHER":
		*e = V3PaymentStatusEnum(v)
		return nil
	default:
		return fmt.Errorf("invalid value for V3PaymentStatusEnum: %v", v)
	}
}
