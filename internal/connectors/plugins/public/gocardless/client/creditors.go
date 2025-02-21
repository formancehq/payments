package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) GetCreditors(ctx context.Context, pageSize int, after string, before string) ([]GocardlessUser, Cursor, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_creditors")

	creditorsResponse, err := c.service.Creditors.List(ctx, gocardless.CreditorListParams{
		Limit:  pageSize,
		After:  after,
		Before: before,
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
			CreatedAt:          parsedTime.Unix(),
			Name:               creditor.Name,
			CountryCode:        creditor.CountryCode,
			PostalCode:         creditor.PostalCode,
			Region:             creditor.Region,
			SchemeIdentifiers:  creditor.SchemeIdentifiers,
			VerificationStatus: creditor.VerificationStatus,
			City:               creditor.City,
			AddressLine1:       creditor.AddressLine1,
			AddressLine2:       creditor.AddressLine2,
			AddressLine3:       creditor.AddressLine3,
		})
	}

	nextCursor := Cursor{
		After:  creditorsResponse.Meta.Cursors.After,
		Before: creditorsResponse.Meta.Cursors.Before,
	}

	return creditors, nextCursor, nil
}
