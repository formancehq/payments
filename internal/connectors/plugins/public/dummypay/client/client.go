package client

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type Client interface {
	FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error)
	FetchBalance(ctx context.Context, accountID string) (*models.PSPBalance, error)
	CreateTransfer(ctx context.Context, paymentInit models.PSPPaymentInitiation) (*models.PSPPayment, error)
}

type client struct {
	directory string
}

func New(dir string) Client {
	return &client{
		directory: dir,
	}
}

func (c *client) FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error) {
	b, err := c.readFile("accounts.json")
	if err != nil {
		return []models.PSPAccount{}, 0, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	accounts := make([]Account, 0)
	err = json.Unmarshal(b, &accounts)
	if err != nil {
		return []models.PSPAccount{}, 0, fmt.Errorf("failed to unmarshal accounts: %w", err)
	}

	next := -1
	pspAccounts := make([]models.PSPAccount, 0, pageSize)
	for i := startToken; i < len(accounts); i++ {
		if len(pspAccounts) >= pageSize {
			if len(accounts)-startToken > len(pspAccounts) {
				next = i
			}
			break
		}

		account := accounts[i]
		pspAccounts = append(pspAccounts, models.PSPAccount{
			Reference:    account.ID,
			CreatedAt:    account.OpeningDate,
			Name:         &account.Name,
			DefaultAsset: &account.Currency,
		})
	}
	return pspAccounts, next, nil
}

func (c *client) FetchBalance(ctx context.Context, accountID string) (*models.PSPBalance, error) {
	b, err := c.readFile("balances.json")
	if err != nil {
		return &models.PSPBalance{}, fmt.Errorf("failed to fetch balances: %w", err)
	}

	balances := make([]Balance, 0)
	err = json.Unmarshal(b, &balances)
	if err != nil {
		return &models.PSPBalance{}, fmt.Errorf("failed to unmarshal balances: %w", err)
	}

	for _, balance := range balances {
		if balance.AccountID != accountID {
			continue
		}
		return &models.PSPBalance{
			AccountReference: balance.AccountID,
			CreatedAt:        time.Now().Truncate(time.Second),
			Asset:            balance.Currency,
			Amount:           big.NewInt(balance.AmountInMinors),
		}, nil
	}
	return &models.PSPBalance{}, nil
}

func (c *client) CreateTransfer(ctx context.Context, paymentInit models.PSPPaymentInitiation) (*models.PSPPayment, error) {
	balance := Balance{
		AccountID:      paymentInit.SourceAccount.Reference,
		AmountInMinors: paymentInit.Amount.Int64(),
		Currency:       paymentInit.Asset,
	}
	b, err := json.Marshal(&balance)
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
		Type:                        models.PAYMENT_TYPE_TRANSFER,
		Status:                      models.PAYMENT_STATUS_SUCCEEDED,
		Scheme:                      models.PAYMENT_SCHEME_OTHER,
		SourceAccountReference:      &paymentInit.SourceAccount.Reference,
		DestinationAccountReference: &paymentInit.DestinationAccount.Reference,
	}, nil
}

func (c *client) writeFile(filename string, b []byte) error {
	filePath := path.Join(c.directory, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to open %q for write: %w", filePath, err)
	}
	defer file.Close()

	_, err = file.Write(b)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filePath, err)
	}
	return nil
}

func (c *client) readFile(filename string) (b []byte, err error) {
	filePath := path.Join(c.directory, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return b, fmt.Errorf("failed to open %q for read: %w", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return b, fmt.Errorf("failed to stat file %q: %w", filePath, err)
	}

	buf := make([]byte, fileInfo.Size())
	_, err = file.Read(buf)
	if err != nil {
		return b, fmt.Errorf("failed to read file %q: %w", filePath, err)
	}
	return buf, nil
}
