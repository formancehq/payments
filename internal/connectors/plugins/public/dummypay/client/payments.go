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

func (c *client) CreatePayment(
	ctx context.Context,
	paymentType models.PaymentType,
	paymentInit models.PSPPaymentInitiation,
) (*models.PSPPayment, error) {
	balances := []Balance{
		{
			AccountID:      paymentInit.SourceAccount.Reference,
			AmountInMinors: paymentInit.Amount.Int64(),
			Currency:       paymentInit.Asset,
		},
	}
	b, err := json.Marshal(&balances)
	if err != nil {
		return &models.PSPPayment{}, fmt.Errorf("failed to marshal new balance: %w", err)
	}

	err = c.writeFile("balances.json", b)
	if err != nil {
		return &models.PSPPayment{}, fmt.Errorf("failed to write balance: %w", err)
	}

	return &models.PSPPayment{
		Reference:                   paymentInit.Reference,
		CreatedAt:                   paymentInit.CreatedAt,
		Amount:                      paymentInit.Amount,
		Asset:                       paymentInit.Asset,
		Type:                        paymentType,
		Status:                      models.PAYMENT_STATUS_SUCCEEDED,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		SourceAccountReference:      &paymentInit.SourceAccount.Reference,
		DestinationAccountReference: &paymentInit.DestinationAccount.Reference,
	}, nil
}

func (c *client) ReversePayment(
	ctx context.Context,
	paymentType models.PaymentType,
	reversal models.PSPPaymentInitiationReversal,
) (models.PSPPayment, error) {
	b, err := c.readFile("balances.json")
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to fetch balances: %w", err)
	}

	balances := make([]Balance, 0)
	if len(b) == 0 {
		return models.PSPPayment{}, fmt.Errorf("no balance data found")
	}

	err = json.Unmarshal(b, &balances)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to unmarshal balances: %w", err)
	}

	var balanceUpdated bool
	for _, balance := range balances {
		if balance.AccountID == reversal.RelatedPaymentInitiation.SourceAccount.Reference {
			if balance.AmountInMinors-reversal.Amount.Int64() < 0 {
				return models.PSPPayment{}, fmt.Errorf("balance will be negative if %d is subtracted", reversal.Amount.Int64())
			}
			balance.AmountInMinors = balance.AmountInMinors - reversal.Amount.Int64()
			balanceUpdated = true
			break
		}
	}

	if !balanceUpdated {
		return models.PSPPayment{}, fmt.Errorf("no reversable balance found")
	}

	b, err = json.Marshal(&balances)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to marshal new balance: %w", err)
	}

	err = c.writeFile("balances.json", b)
	if err != nil {
		return models.PSPPayment{}, fmt.Errorf("failed to write balance: %w", err)
	}

	return models.PSPPayment{
		Reference:                   reversal.Reference,
		CreatedAt:                   reversal.CreatedAt,
		Amount:                      reversal.Amount,
		Asset:                       reversal.Asset,
		Type:                        paymentType,
		Status:                      models.PAYMENT_STATUS_REFUNDED,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		SourceAccountReference:      &reversal.RelatedPaymentInitiation.SourceAccount.Reference,
		DestinationAccountReference: &reversal.RelatedPaymentInitiation.DestinationAccount.Reference,
	}, nil
}
