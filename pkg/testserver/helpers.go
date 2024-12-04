package testserver

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	dummy "github.com/formancehq/payments/internal/connectors/plugins/public/dummypay/client"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	. "github.com/formancehq/go-libs/v2/testing/utils"
	. "github.com/onsi/ginkgo/v2"
)

func NewTestServer(configurationProvider func() Configuration) *Deferred[*Server] {
	d := NewDeferred[*Server]()
	BeforeEach(func() {
		d.Reset()
		d.SetValue(New(GinkgoT(), configurationProvider()))
	})
	return d
}

func Subscribe(t T, testServer *Server) chan *nats.Msg {
	subscription, ch, err := testServer.Subscribe()
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, subscription.Unsubscribe())
	})

	return ch
}

func GeneratePSPData(dir string) error {
	num := 10
	_, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("path %q does not exist: %w", dir, err)
	}

	accounts := make([]dummy.Account, 0, num)
	balances := make([]dummy.Balance, 0, num)
	startTime := time.Now().Truncate(time.Second)
	for i := 0; i < num; i++ {
		id := uuid.New().String()
		accounts = append(accounts, dummy.Account{
			ID:          id,
			Name:        fmt.Sprintf("dummy-account-%d", i),
			Currency:    "EUR",
			OpeningDate: startTime.Add(-time.Duration(i) * time.Minute),
		})
		balances = append(balances, dummy.Balance{
			AccountID:      id,
			AmountInMinors: int64(i*100 + 23),
			Currency:       "EUR",
		})
	}

	accountsFilePath := path.Join(dir, "accounts.json")
	err = persistData(accountsFilePath, accounts)
	if err != nil {
		return err
	}
	balancesFilePath := path.Join(dir, "balances.json")
	err = persistData(balancesFilePath, balances)
	if err != nil {
		return err
	}
	return nil
}

func persistData(filePath string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for %s: %w", filePath, err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create %q: %w", filePath, err)
	}
	defer file.Close()

	if _, err := file.Write(b); err != nil {
		return fmt.Errorf("failed to write to %q: %w", filePath, err)
	}
	return nil
}
