package tink

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var from models.BankBridgeFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	var webhook client.AccountCreatedWebhook
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	resp, err := p.client.ListTransactions(ctx, client.ListTransactionRequest{
		UserID:        webhook.ExternalUserID,
		AccountID:     webhook.ID,
		PageSize:      req.PageSize,
		NextPageToken: "",
	})
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	fmt.Println("TOTOTOTO", resp.NextPageToken)
	for _, transaction := range resp.Transactions {
		fmt.Println("TOTOTOTO", transaction)
	}

	return models.FetchNextPaymentsResponse{
		Payments:         []models.PSPPayment{},
		PaymentsToDelete: []models.PSPPayment{},
		NewState:         []byte{},
		HasMore:          false,
	}, nil
}
