package client

import (
	"context"
	"time"

	"github.com/get-momo/atlar-v1-go-client/client/credit_transfers"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

func (c *client) PostV1CreditTransfers(ctx context.Context, req *atlar_models.CreatePaymentRequest) (*credit_transfers.PostV1CreditTransfersCreated, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "create_credit_transfer")

	postCreditTransfersParams := credit_transfers.PostV1CreditTransfersParams{
		Context:        ctx,
		CreditTransfer: req,
	}

	resp, err := c.client.CreditTransfers.PostV1CreditTransfers(&postCreditTransfersParams)
	return resp, wrapSDKErr(err)
}

func (c *client) GetV1CreditTransfersGetByExternalIDExternalID(ctx context.Context, externalID string) (*credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_credit_transfer")

	getCreditTransferParams := credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDParams{
		Context:    ctx,
		ExternalID: externalID,
	}

	resp, err := c.client.CreditTransfers.GetV1CreditTransfersGetByExternalIDExternalID(&getCreditTransferParams)
	return resp, wrapSDKErr(err)
}
