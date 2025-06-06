// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

type V3ReversePaymentInitiationResponseData struct {
	// Since this call is asynchronous, the response will contain the ID of the task that was created to reverse the payment initiation. You can use the task API to check the status of the task and get the resulting payment ID.
	//
	TaskID *string `json:"taskID,omitempty"`
	// Related payment initiation reversal object ID created.
	//
	PaymentInitiationReversalID *string `json:"paymentInitiationReversalID,omitempty"`
}

func (o *V3ReversePaymentInitiationResponseData) GetTaskID() *string {
	if o == nil {
		return nil
	}
	return o.TaskID
}

func (o *V3ReversePaymentInitiationResponseData) GetPaymentInitiationReversalID() *string {
	if o == nil {
		return nil
	}
	return o.PaymentInitiationReversalID
}

type V3ReversePaymentInitiationResponse struct {
	Data V3ReversePaymentInitiationResponseData `json:"data"`
}

func (o *V3ReversePaymentInitiationResponse) GetData() V3ReversePaymentInitiationResponseData {
	if o == nil {
		return V3ReversePaymentInitiationResponseData{}
	}
	return o.Data
}
