package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
	"github.com/google/uuid"
)

type Quote struct {
	ID uuid.UUID `json:"id"`
}

func (c *client) CreateQuote(ctx context.Context, profileID, currency string, amount json.Number) (Quote, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_quote")

	var quote Quote

	reqBody, err := json.Marshal(map[string]interface{}{
		"sourceCurrency": currency,
		"targetCurrency": currency,
		"sourceAmount":   amount,
	})
	if err != nil {
		return quote, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.endpoint("v3/profiles/"+profileID+"/quotes"),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return quote, err
	}
	req.Header.Set("Content-Type", "application/json")

	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &quote, &errRes)
	if err != nil {
		return Quote{}, errorsutils.NewWrappedError(
			fmt.Errorf("failed to create quote: %v", errRes.Error(statusCode)),
			err,
		)
	}
	return quote, nil
}
