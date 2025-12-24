package client

import (
	"context"
	"strings"

	cbtx "github.com/coinbase-samples/prime-sdk-go/transactions"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Transaction struct {
	ID       string
	Symbol   string
	Amount   string
	Type     string
	Status   string
	FromType string
	ToType   string
	WalletID string
}

func (c *client) GetPortfolioTransactions(ctx context.Context, portfolioId string, page, pageSize int) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_portfolio_transactions")
	svc := cbtx.NewTransactionsService(c.sdk)
	res, err := svc.ListPortfolioTransactions(ctx, &cbtx.ListPortfolioTransactionsRequest{PortfolioId: portfolioId})
	if err != nil {
		return nil, err
	}
	all := make([]*Transaction, 0, len(res.Transactions))
	for _, t := range res.Transactions {
		all = append(all, &Transaction{
			ID:       t.Id,
			Symbol:   strings.ToUpper(t.Symbol),
			Amount:   t.Amount,
			Type:     string(t.Type),
			Status:   string(t.Status),
			FromType: string(t.TransferFrom.Type),
			ToType:   string(t.TransferTo.Type),
			WalletID: t.WalletId,
		})
	}
	if pageSize <= 0 {
		return all, nil
	}
	start := page * pageSize
	if start >= len(all) {
		return []*Transaction{}, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], nil
}

func (c *client) GetWalletTransactions(ctx context.Context, portfolioId string, walletId string, page, pageSize int) ([]*Transaction, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_wallet_transactions")
	svc := cbtx.NewTransactionsService(c.sdk)
	res, err := svc.ListWalletTransactions(ctx, &cbtx.ListWalletTransactionsRequest{PortfolioId: portfolioId, WalletId: walletId})
	if err != nil {
		return nil, err
	}
	all := make([]*Transaction, 0, len(res.Transactions))
	for _, t := range res.Transactions {
		all = append(all, &Transaction{
			ID:       t.Id,
			Symbol:   strings.ToUpper(t.Symbol),
			Amount:   t.Amount,
			Type:     string(t.Type),
			Status:   string(t.Status),
			FromType: string(t.TransferFrom.Type),
			ToType:   string(t.TransferTo.Type),
			WalletID: t.WalletId,
		})
	}
	if pageSize <= 0 {
		return all, nil
	}
	start := page * pageSize
	if start >= len(all) {
		return []*Transaction{}, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], nil
}
