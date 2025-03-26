package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
	errorsutils "github.com/formancehq/payments/internal/utils/errors"
)

type Profile struct {
	ID   uint64 `json:"id"`
	Type string `json:"type"`
}

func (c *client) GetProfiles(ctx context.Context) ([]Profile, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_profiles")

	var profiles []Profile
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("v2/profiles"), http.NoBody)
	if err != nil {
		return profiles, err
	}

	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(ctx, req, &profiles, &errRes)
	if err != nil {
		return nil, errorsutils.NewWrappedError(
			fmt.Errorf("failed to get profiles: %v", errRes.Error(statusCode)),
			err,
		)
	}
	return profiles, nil
}
