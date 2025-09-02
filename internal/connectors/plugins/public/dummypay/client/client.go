package client

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/formancehq/payments/internal/models"
)

type Client interface {
	FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error)
	FetchBalance(ctx context.Context, accountID string) (*models.PSPBalance, error)
	CreatePayment(ctx context.Context, paymentType models.PaymentType, paymentInit models.PSPPaymentInitiation) (*models.PSPPayment, error)
	ReversePayment(ctx context.Context, paymentType models.PaymentType, reversal models.PSPPaymentInitiationReversal) (models.PSPPayment, error)
	CreateUser(ctx context.Context, user models.PSPPaymentServiceUser) (string, error)
	DeleteUser(ctx context.Context, userID string) error
	CompleteLink(ctx context.Context, userID string, connectionID string) error
	DeleteUserConnection(ctx context.Context, userID string, connectionID string) error
}

type client struct {
	directory string
}

func New(dir string) Client {
	return &client{
		directory: dir,
	}
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
		if os.IsNotExist(err) {
			return b, nil
		}
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

func (c *client) deleteFile(filename string) error {
	filePath := path.Join(c.directory, filename)
	return os.Remove(filePath)
}
