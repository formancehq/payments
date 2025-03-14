package client

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	gocardless "github.com/gocardless/gocardless-pro-go/v4"
)

func (c *client) GetCustomers(ctx context.Context, pageSize int, after string) ([]GocardlessUser, Cursor, error) {

	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_customers")

	customersResponse, err := c.service.GetGocardlessCustomers(ctx, gocardless.CustomerListParams{
		Limit: pageSize,
		After: after,
	})

	if err != nil {
		return []GocardlessUser{}, Cursor{}, err
	}

	var customers []GocardlessUser

	for _, customer := range customersResponse.Customers {
		parsedTime, err := time.Parse(time.RFC3339Nano, customer.CreatedAt)
		if err != nil {
			return []GocardlessUser{}, Cursor{}, fmt.Errorf("failed to parse creation time: %w", err)
		}

		customers = append(customers, GocardlessUser{
			Id:                    customer.Id,
			CreatedAt:             parsedTime,
			Name:                  customer.GivenName + " " + customer.FamilyName,
			CountryCode:           customer.CountryCode,
			PostalCode:            customer.PostalCode,
			Region:                customer.Region,
			City:                  customer.City,
			AddressLine1:          customer.AddressLine1,
			AddressLine2:          customer.AddressLine2,
			AddressLine3:          customer.AddressLine3,
			CompanyName:           customer.CompanyName,
			PhoneNumber:           customer.PhoneNumber,
			Email:                 customer.Email,
			Language:              customer.Language,
			Metadata:              customer.Metadata,
			SwedishIdentityNumber: customer.SwedishIdentityNumber,
			DanishIdentityNumber:  customer.DanishIdentityNumber,
		})
	}

	nextCursor := Cursor{
		After: customersResponse.Meta.Cursors.After,
	}

	return customers, nextCursor, nil
}
