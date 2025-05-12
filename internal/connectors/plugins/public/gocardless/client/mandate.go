package client

import (
	"context"

	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) GetMandate(ctx context.Context, mandateId string) (*gocardless.Mandate, error) {

	mandate, err := c.service.GetMandate(ctx, mandateId)

	if err != nil {
		return nil, err
	}

	return mandate, nil
}
