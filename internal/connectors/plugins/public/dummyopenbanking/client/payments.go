package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type Payment struct {
	Reference string    `json:"reference"`
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (c *client) FetchPayments(ctx context.Context, startToken int, pageSize int) ([]models.PSPPayment, int, error) {
	b, err := c.readFile("payments.json")
	if err != nil {
		return []models.PSPPayment{}, 0, fmt.Errorf("failed to fetch payments: %w", err)
	}

	pspPayments := make([]models.PSPPayment, 0, pageSize)
	if len(b) == 0 {
		return pspPayments, -1, nil
	}

	payments := make([]Payment, 0)
	err = json.Unmarshal(b, &payments)
	if err != nil {
		return []models.PSPPayment{}, 0, fmt.Errorf("failed to unmarshal payments: %w", err)
	}

	next := -1
	for i := startToken; i < len(payments); i++ {
		if len(pspPayments) >= pageSize {
			if len(payments)-startToken > len(pspPayments) {
				next = i
			}
			break
		}

		payment := payments[i]
		status, err := models.PaymentStatusFromString(payment.Status)
		if err != nil {
			return []models.PSPPayment{}, 0, fmt.Errorf("failed to parse payment status: %w", err)
		}

		paymentType, err := models.PaymentTypeFromString(payment.Type)
		if err != nil {
			return []models.PSPPayment{}, 0, fmt.Errorf("failed to parse payment type: %w", err)
		}

		pspPayments = append(pspPayments, models.PSPPayment{
			Reference: payment.Reference,
			Amount:    big.NewInt(payment.Amount),
			Asset:     payment.Currency,
			Status:    status,
			CreatedAt: payment.CreatedAt,
			Type:      paymentType,
		})
	}

	return pspPayments, next, nil
}
