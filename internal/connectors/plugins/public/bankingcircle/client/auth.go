package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

func (c *client) login(ctx context.Context) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "authenticate")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.authorizationEndpoint+"/api/v1/authorizations/authorize", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	//nolint:tagliatelle // allow for client-side structures
	type response struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   string `json:"expires_in"`
	}
	type responseError struct {
		ErrorCode string `json:"errorCode"`
		ErrorText string `json:"errorText"`
	}

	var res response
	var errors []responseError
	statusCode, err := c.httpClient.Do(ctx, req, &res, &errors)
	if err != nil {
		if len(errors) > 0 {
			return errorsutils.NewWrappedError(
				fmt.Errorf("failed to login, status code %d: %s", statusCode, errors[0].ErrorText),
				err,
			)
		}
		return errorsutils.NewWrappedError(
			fmt.Errorf("failed to login, status code %d", statusCode),
			err,
		)
	}

	c.accessToken = res.AccessToken
	expiresIn, err := strconv.Atoi(res.ExpiresIn)
	if err != nil {
		return fmt.Errorf("failed to convert expires_in to int: %w", err)
	}
	c.accessTokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return nil
}

func (c *client) ensureAccessTokenIsValid(ctx context.Context) error {
	if c.accessToken == "" {
		return c.login(ctx)
	}

	if c.accessTokenExpiresAt.After(time.Now().Add(5 * time.Second)) {
		return nil
	}

	return c.login(ctx)
}
