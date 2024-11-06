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

	return c.client.CreditTransfers.PostV1CreditTransfers(&postCreditTransfersParams)

}

func (c *client) GetV1CreditTransfersGetByExternalIDExternalID(ctx context.Context, externalID string) (*credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "get_credit_transfer")

	getCreditTransferParams := credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDParams{
		Context:    ctx,
		ExternalID: externalID,
	}

	return c.client.CreditTransfers.GetV1CreditTransfersGetByExternalIDExternalID(&getCreditTransferParams)
}
