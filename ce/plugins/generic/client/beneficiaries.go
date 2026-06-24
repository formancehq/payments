package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/formancehq/payments/genericclient/v3"
	"github.com/formancehq/payments/pkg/domain/metrics"
)

func (c *client) ListBeneficiaries(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Beneficiary, error) {
	ctx = metrics.OperationContext(ctx, "list_beneficiaries")

	u, err := url.Parse(fmt.Sprintf("%s/beneficiaries", c.baseURL))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("page", strconv.FormatInt(page, 10))
	q.Set("pageSize", strconv.FormatInt(pageSize, 10))
	q.Set("sort", "createdAt:asc")
	if !createdAtFrom.IsZero() {
		q.Set("createdAtFrom", createdAtFrom.UTC().Format(time.RFC3339))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	var beneficiaries []genericclient.Beneficiary
	var errResp genericAPIError
	if _, err = c.httpClient.Do(ctx, req, &beneficiaries, &errResp); err != nil {
		return nil, fmt.Errorf("failed to list beneficiaries: %w", err)
	}
	return beneficiaries, nil
}
