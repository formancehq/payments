package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

func (c *client) FetchTrades(ctx context.Context) ([]models.PSPTrade, error) {
	b, err := c.readFile("trades.json")
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return []models.PSPTrade{}, nil
	}

	var trades []models.PSPTrade
	if err := json.Unmarshal(b, &trades); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trades: %w", err)
	}

	return trades, nil
}
