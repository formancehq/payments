package increase

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

const (
	PayoutTypeMetadataKey = "payout_type"
	PayoutTypeACH        = "ach"
	PayoutTypeWire       = "wire"
	PayoutTypeCheck      = "check"
	PayoutTypeRTP        = "rtp"

	// Standard Entry Class Codes for ACH transfers
	SECCodePPD = "ppd" // Prearranged Payment and Deposit Entry
	SECCodeCCD = "ccd" // Corporate Credit or Debit Entry
)

func (p *Plugin) CreateTransfer(ctx context.Context, req models.CreateTransferRequest) (models.CreateTransferResponse, error) {
	createReq := &client.CreateTransferRequest{
		AccountID:     req.PaymentInitiation.SourceAccountID,
		Amount:        req.PaymentInitiation.Amount,
		Description:   req.PaymentInitiation.Description,
	}

	transfer, err := p.client.CreateTransfer(ctx, createReq)
	if err != nil {
		return models.CreateTransferResponse{}, fmt.Errorf("failed to create transfer: %w", err)
	}

	raw, err := json.Marshal(transfer)
	if err != nil {
		return models.CreateTransferResponse{}, fmt.Errorf("failed to marshal transfer: %w", err)
	}

	payment := &models.PSPPayment{
		ID:        transfer.ID,
		CreatedAt: transfer.CreatedAt,
		Reference: transfer.ID,
		Type:      models.PaymentType(transfer.Type),
		Status:    models.PaymentStatus(transfer.Status),
		Amount:    transfer.Amount,
		Currency:  transfer.Currency,
		Raw:       raw,
	}

	return models.CreateTransferResponse{
		Payment: payment,
	}, nil
}

func (p *Plugin) determinePayoutType(metadata map[string]string) (string, error) {
	payoutType, ok := metadata[PayoutTypeMetadataKey]
	if !ok {
		return "", fmt.Errorf("missing payout type in metadata")
	}

	switch payoutType {
	case PayoutTypeACH:
		return PayoutTypeACH, nil
	case PayoutTypeWire:
		return PayoutTypeWire, nil
	case PayoutTypeCheck:
		return PayoutTypeCheck, nil
	case PayoutTypeRTP:
		return PayoutTypeRTP, nil
	default:
		return "", fmt.Errorf("unsupported payout type: %s", payoutType)
	}
}

func (p *Plugin) CreatePayout(ctx context.Context, req models.CreatePayoutRequest) (models.CreatePayoutResponse, error) {
	payoutType, err := p.determinePayoutType(req.PaymentInitiation.Metadata)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("failed to determine payout type: %w", err)
	}

	var (
		transfer *client.Transfer
		err      error
	)

	switch payoutType {
	case PayoutTypeACH:
		secCode := SECCodePPD // Default to PPD for consumer transfers
		if businessType, ok := req.PaymentInitiation.Metadata["business_type"]; ok && businessType == "business" {
			secCode = SECCodeCCD // Use CCD for business transfers
		}

		createReq := &client.CreateACHTransferRequest{
			CreateTransferRequest: client.CreateTransferRequest{
				AccountID:     req.PaymentInitiation.SourceAccountID,
				Amount:        req.PaymentInitiation.Amount,
				Description:   req.PaymentInitiation.Description,
			},
			StandardEntryClassCode: secCode,
		}
		transfer, err = p.client.CreateACHTransfer(ctx, createReq)

	case PayoutTypeWire:
		createReq := &client.CreateWireTransferRequest{
			CreateTransferRequest: client.CreateTransferRequest{
				AccountID:     req.PaymentInitiation.SourceAccountID,
				Amount:        req.PaymentInitiation.Amount,
				Description:   req.PaymentInitiation.Description,
			},
			MessageToRecipient: req.PaymentInitiation.Description,
		}
		transfer, err = p.client.CreateWireTransfer(ctx, createReq)

	case PayoutTypeCheck:
		memo := req.PaymentInitiation.Description
		if checkMemo, ok := req.PaymentInitiation.Metadata["check_memo"]; ok {
			memo = checkMemo
		}

		createReq := &client.CreateCheckTransferRequest{
			CreateTransferRequest: client.CreateTransferRequest{
				AccountID:     req.PaymentInitiation.SourceAccountID,
				Amount:        req.PaymentInitiation.Amount,
				Description:   req.PaymentInitiation.Description,
			},
			PhysicalCheck: client.PhysicalCheck{
				Memo: memo,
			},
		}
		transfer, err = p.client.CreateCheckTransfer(ctx, createReq)

	case PayoutTypeRTP:
		createReq := &client.CreateRTPTransferRequest{
			CreateTransferRequest: client.CreateTransferRequest{
				AccountID:     req.PaymentInitiation.SourceAccountID,
				Amount:        req.PaymentInitiation.Amount,
				Description:   req.PaymentInitiation.Description,
			},
		}
		transfer, err = p.client.CreateRTPTransfer(ctx, createReq)

	default:
		return models.CreatePayoutResponse{}, fmt.Errorf("unsupported payout type: %s", payoutType)
	}

	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("failed to create payout: %w", err)
	}

	raw, err := json.Marshal(transfer)
	if err != nil {
		return models.CreatePayoutResponse{}, fmt.Errorf("failed to marshal transfer: %w", err)
	}

	payment := &models.PSPPayment{
		ID:        transfer.ID,
		CreatedAt: transfer.CreatedAt,
		Reference: transfer.ID,
		Type:      models.PaymentType(transfer.Type),
		Status:    models.PaymentStatus(transfer.Status),
		Amount:    transfer.Amount,
		Currency:  transfer.Currency,
		Raw:       raw,
	}

	return models.CreatePayoutResponse{
		Payment: payment,
	}, nil
}
