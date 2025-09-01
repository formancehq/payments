package client

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	FetchAccounts(ctx context.Context, startToken int, pageSize int) ([]models.PSPAccount, int, error)
	FetchPayments(ctx context.Context, startToken int, pageSize int) ([]models.PSPPayment, int, error)
	CreateUser(ctx context.Context, user models.PSPPaymentServiceUser) (string, error)
	DeleteUser(ctx context.Context, userID string) error
	CompleteLink(ctx context.Context, userID string, connectionID string) error
	DeleteUserConnection(ctx context.Context, userID string, connectionID string) error
}

type client struct {
	directory string

	formanceHTTPClient    httpwrapper.Client
	formanceStackEndpoint string

	connectorID models.ConnectorID
}

func New(name string, dir string, connectorID models.ConnectorID) (Client, error) {
	formanceStackEndpoint, err := url.JoinPath(os.Getenv("STACK_PUBLIC_URL"), "api", "payments", "v3")
	if err != nil {
		return nil, err
	}

	formanceHTTPClient := httpwrapper.NewClient(&httpwrapper.Config{
		Transport: metrics.NewTransport(name, metrics.TransportOpts{}),
	})

	return &client{
		directory:             dir,
		formanceStackEndpoint: formanceStackEndpoint,
		connectorID:           connectorID,
		formanceHTTPClient:    formanceHTTPClient,
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
