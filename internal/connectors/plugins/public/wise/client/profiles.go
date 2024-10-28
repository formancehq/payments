package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
)

type Profile struct {
	ID   uint64 `json:"id"`
	Type string `json:"type"`
}

func (c *client) GetProfiles(ctx context.Context) ([]Profile, error) {
	ctx = context.WithValue(ctx, httpwrapper.MetricOperationContextKey, "list_profiles")

	var profiles []Profile
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("v2/profiles"), http.NoBody)
	if err != nil {
		return profiles, err
	}

	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &profiles, &errRes)
	if err != nil {
		return profiles, fmt.Errorf("failed to make profiles: %w %w", err, errRes.Error(statusCode).Error())
	}
	return profiles, nil
}
