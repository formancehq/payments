package client

import (
	"context"
	"fmt"
)

func (c *client) DeleteUserConnection(ctx context.Context, userID string, connectionID string) error {
	return c.deleteFile(fmt.Sprintf("complete-link-%s-%s.json", userID, connectionID))
}
