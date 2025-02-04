package client

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/Increase/increase-go"
)

type Transfer struct {
	ID          string    `json:"id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	AccountID   string    `json:"account_id"`
	Description string    `json:"description"`
}

type CreateTransferRequest struct {
	AccountID          string `json:"account_id"`
	Amount            int64  `json:"amount"`
	Description       string `json:"description"`
	RequireApproval   bool   `json:"require_approval"`
	ExternalAccountID string `json:"external_account_id"`
}

type CreateACHTransferRequest struct {
	CreateTransferRequest
	CompanyDescriptiveDate string `json:"company_descriptive_date"`
	StandardEntryClassCode string `json:"standard_entry_class_code"`
}

type CreateWireTransferRequest struct {
	CreateTransferRequest
	MessageToRecipient string `json:"message_to_recipient"`
}

type CreateCheckTransferRequest struct {
	CreateTransferRequest
	PhysicalCheck PhysicalCheck `json:"physical_check"`
}

type CreateRTPTransferRequest struct {
	CreateTransferRequest
}

type PhysicalCheck struct {
	Memo string `json:"memo"`
}

func mapTransfer(t *increase.Transfer) *Transfer {
	return &Transfer{
		ID:          t.ID,
		Amount:      t.Amount,
		Currency:    string(t.Currency),
		Status:      string(t.Status),
		Type:        string(t.Type),
		CreatedAt:   t.CreatedAt,
		AccountID:   t.AccountID,
		Description: t.Description,
	}
}

func (c *client) CreateTransfer(ctx context.Context, req *CreateTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_transfer")

	params := &increase.AccountTransferCreateParams{
		AccountID:        req.AccountID,
		Amount:          req.Amount,
		Description:     increase.F(req.Description),
		RequireApproval: increase.F(req.RequireApproval),
	}

	transfer, err := c.sdk.AccountTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapTransfer(transfer), nil
}

func (c *client) CreateACHTransfer(ctx context.Context, req *CreateACHTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_ach_transfer")

	params := &increase.ACHTransferCreateParams{
		AccountID:              req.AccountID,
		Amount:                req.Amount,
		ExternalAccountID:     req.ExternalAccountID,
		RequireApproval:       increase.F(req.RequireApproval),
		CompanyDescriptiveDate: increase.F(req.CompanyDescriptiveDate),
		StandardEntryClassCode: increase.F(req.StandardEntryClassCode),
	}

	transfer, err := c.sdk.ACHTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapTransfer(transfer), nil
}

func (c *client) CreateWireTransfer(ctx context.Context, req *CreateWireTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_wire_transfer")

	params := &increase.WireTransferCreateParams{
		AccountID:          req.AccountID,
		Amount:            req.Amount,
		ExternalAccountID: req.ExternalAccountID,
		RequireApproval:   increase.F(req.RequireApproval),
		MessageToRecipient: increase.F(req.MessageToRecipient),
	}

	transfer, err := c.sdk.WireTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapTransfer(transfer), nil
}

func (c *client) CreateCheckTransfer(ctx context.Context, req *CreateCheckTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_check_transfer")

	params := &increase.CheckTransferCreateParams{
		AccountID:          req.AccountID,
		Amount:            req.Amount,
		ExternalAccountID: req.ExternalAccountID,
		RequireApproval:   increase.F(req.RequireApproval),
		PhysicalCheck: &increase.CheckTransferCreateParamsPhysicalCheck{
			Memo: increase.F(req.PhysicalCheck.Memo),
		},
	}

	transfer, err := c.sdk.CheckTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapTransfer(transfer), nil
}

func (c *client) CreateRTPTransfer(ctx context.Context, req *CreateRTPTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_rtp_transfer")

	params := &increase.RealTimePaymentsTransferCreateParams{
		AccountID:          req.AccountID,
		Amount:            req.Amount,
		ExternalAccountID: req.ExternalAccountID,
		RequireApproval:   increase.F(req.RequireApproval),
	}

	transfer, err := c.sdk.RealTimePaymentsTransfers.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapTransfer(transfer), nil
}
