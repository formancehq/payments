package client

import (
	"context"
	"encoding/json"

	cbtx "github.com/coinbase-samples/prime-sdk-go/transactions"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type TransferRequest struct {
	PortfolioID         string
	WalletID            string
	DestinationWalletID string
	Amount              string
	CurrencySymbol      string
	IdempotencyKey      string
}

type TransferResponse struct {
	ID             string
	Symbol         string
	Amount         string
	Status         string
	FromWalletID   string
	ToWalletID     string
	IdempotencyKey string
	Raw            json.RawMessage
}

func (c *client) InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "initiate_transfer")

	svc := cbtx.NewTransactionsService(c.sdk)
	res, err := svc.CreateWalletTransfer(ctx, &cbtx.CreateWalletTransferRequest{
		PortfolioId:         tr.PortfolioID,
		SourceWalletId:      tr.WalletID,
		Amount:              tr.Amount,
		DestinationWalletId: tr.DestinationWalletID,
		IdempotencyKey:      tr.IdempotencyKey,
		Symbol:              tr.CurrencySymbol,
	})
	if err != nil {
		return nil, err
	}

	raw, _ := json.Marshal(res)
	return &TransferResponse{
		Symbol:         tr.CurrencySymbol,
		Amount:         tr.Amount,
		FromWalletID:   tr.WalletID,
		ToWalletID:     tr.DestinationWalletID,
		IdempotencyKey: tr.IdempotencyKey,
		Raw:            raw,
	}, nil
}
