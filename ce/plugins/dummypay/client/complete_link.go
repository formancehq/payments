package client

import (
	"context"
	"encoding/json"
	"fmt"
)

type CompleteLink struct {
	UserID       string
	ConnectionID string
}

func (c *client) CompleteLink(ctx context.Context, userID string, connectionID string) error {
	completeLink := CompleteLink{
		UserID:       userID,
		ConnectionID: connectionID,
	}

	b, err := json.Marshal(&completeLink)
	if err != nil {
		return fmt.Errorf("failed to marshal complete link: %w", err)
	}

	err = c.writeFile(fmt.Sprintf("complete-link-%s-%s.json", userID, connectionID), b)
	if err != nil {
		return fmt.Errorf("failed to write complete link: %w", err)
	}

	return nil
}
