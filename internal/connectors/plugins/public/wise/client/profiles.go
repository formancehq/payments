package client

import (
	"context"
	"fmt"
	"net/http"
)

type Profile struct {
	ID   uint64 `json:"id"`
	Type string `json:"type"`
}

func (c *client) GetProfiles(ctx context.Context) ([]Profile, error) {
	// TODO(polo): metrics
	// f := connectors.ClientMetrics(ctx, "wise", "list_profiles")
	// now := time.Now()
	// defer f(ctx, now)

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
