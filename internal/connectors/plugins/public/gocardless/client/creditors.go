package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) GetCreditors(ctx context.Context, pageSize int, after string) ([]GocardlessUser, Cursor, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_creditors")

	creditorsResponse, err := c.service.GetGocardlessCreditors(ctx, gocardless.CreditorListParams{
		Limit: pageSize,
		After: after,
	})

	if err != nil {
		return []GocardlessUser{}, Cursor{}, err
	}

	var creditors []GocardlessUser

	for _, creditor := range creditorsResponse.Creditors {
		parsedTime, err := time.Parse(time.RFC3339Nano, creditor.CreatedAt)
		if err != nil {
			return []GocardlessUser{}, Cursor{}, fmt.Errorf("failed to parse creation time: %w", err)
		}
		creditors = append(creditors, GocardlessUser{
			Id:                 creditor.Id,
			CreatedAt:          parsedTime,
			Name:               creditor.Name,
			CountryCode:        creditor.CountryCode,
			PostalCode:         creditor.PostalCode,
			Region:             creditor.Region,
			VerificationStatus: creditor.VerificationStatus,
			City:               creditor.City,
			AddressLine1:       creditor.AddressLine1,
			AddressLine2:       creditor.AddressLine2,
			AddressLine3:       creditor.AddressLine3,
		})
	}

	nextCursor := Cursor{
		After: creditorsResponse.Meta.Cursors.After,
	}

	return creditors, nextCursor, nil
}
