package client

import (
	"context"
	"fmt"
)

func (c *client) DeleteUser(ctx context.Context, userID string) error {
	return c.deleteFile(fmt.Sprintf("user-%s.json", userID))
}
