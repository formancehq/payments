package client

import (
	"context"
	"time"
)

type Transfer struct {
	ID            string    `json:"id"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	Type          string    `json:"type"`
	CreatedAt     time.Time `json:"created_at"`
	AccountID     string    `json:"account_id"`
	Description   string    `json:"description"`
}

type CreateTransferRequest struct {
	AccountID          string            `json:"account_id"`
	Amount            int64             `json:"amount"`
	Description       string            `json:"description"`
	RequireApproval   bool              `json:"require_approval"`
	ExternalAccountID string            `json:"external_account_id"`
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

func (c *client) CreateTransfer(ctx context.Context, req *CreateTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_transfer")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/account_transfers", body)
	if err != nil {
		return nil, err
	}

	var transfer Transfer
	if err := c.do(httpReq, &transfer); err != nil {
		return nil, err
	}

	return &transfer, nil
}

func (c *client) CreateACHTransfer(ctx context.Context, req *CreateACHTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_ach_transfer")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/ach_transfers", body)
	if err != nil {
		return nil, err
	}

	var transfer Transfer
	if err := c.do(httpReq, &transfer); err != nil {
		return nil, err
	}

	return &transfer, nil
}

func (c *client) CreateWireTransfer(ctx context.Context, req *CreateWireTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_wire_transfer")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/wire_transfers", body)
	if err != nil {
		return nil, err
	}

	var transfer Transfer
	if err := c.do(httpReq, &transfer); err != nil {
		return nil, err
	}

	return &transfer, nil
}

func (c *client) CreateCheckTransfer(ctx context.Context, req *CreateCheckTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_check_transfer")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/check_transfers", body)
	if err != nil {
		return nil, err
	}

	var transfer Transfer
	if err := c.do(httpReq, &transfer); err != nil {
		return nil, err
	}

	return &transfer, nil
}

func (c *client) CreateRTPTransfer(ctx context.Context, req *CreateRTPTransferRequest) (*Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_rtp_transfer")

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(req); err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := c.newRequest(ctx, http.MethodPost, "/real_time_payments_transfers", body)
	if err != nil {
		return nil, err
	}

	var transfer Transfer
	if err := c.do(httpReq, &transfer); err != nil {
		return nil, err
	}

	return &transfer, nil
}
