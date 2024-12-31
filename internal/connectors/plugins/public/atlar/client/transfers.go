package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/get-momo/atlar-v1-go-client/client/credit_transfers"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

func (c *client) PostV1CreditTransfers(ctx context.Context, req *atlar_models.CreatePaymentRequest) (*credit_transfers.PostV1CreditTransfersCreated, error) {
	postCreditTransfersParams := credit_transfers.PostV1CreditTransfersParams{
		Context:        metrics.OperationContext(ctx, "create_credit_transfer"),
		CreditTransfer: req,
		HTTPClient:     c.httpClient,
	}

	resp, err := c.client.CreditTransfers.PostV1CreditTransfers(&postCreditTransfersParams)
	return resp, wrapSDKErr(err)
}

func (c *client) GetV1CreditTransfersGetByExternalIDExternalID(ctx context.Context, externalID string) (*credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK, error) {
	getCreditTransferParams := credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDParams{
		Context:    metrics.OperationContext(ctx, "get_credit_transfer"),
		ExternalID: externalID,
		HTTPClient: c.httpClient,
	}

	resp, err := c.client.CreditTransfers.GetV1CreditTransfersGetByExternalIDExternalID(&getCreditTransferParams)
	return resp, wrapSDKErr(err)
}
