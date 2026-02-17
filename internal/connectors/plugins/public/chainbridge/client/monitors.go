package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type Monitor struct {
	ID        string    `json:"id"`
	Chain     string    `json:"chain"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

func (c *client) GetMonitors(ctx context.Context) ([]*Monitor, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_monitors")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.buildEndpoint("monitors"), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var resp struct {
		Data []*Monitor `json:"data"`
	}
	var errRes chainbridgeError

	_, err = c.httpClient.Do(ctx, req, &resp, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get monitors: %s", errRes.ErrorMessage),
			err,
		)
	}

	return resp.Data, nil
}
