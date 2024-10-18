package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/formancehq/go-libs/v2/errorsutils"
)

type User struct {
	ID           string `json:"Id"`
	CreationDate int64  `json:"CreationDate"`
}

func (c *Client) GetUsers(ctx context.Context, page int, pageSize int) ([]User, error) {
	// TODO(polo): add metrics
	// f := connectors.ClientMetrics(ctx, "mangopay", "list_users")
	// now := time.Now()
	// defer f(ctx, now)

	endpoint := fmt.Sprintf("%s/v2.01/%s/users", c.endpoint, c.clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	q := req.URL.Query()
	q.Add("per_page", strconv.Itoa(pageSize))
	q.Add("page", fmt.Sprint(page))
	q.Add("Sort", "CreationDate:ASC")
	req.URL.RawQuery = q.Encode()

	var users []User
	statusCode, err := c.httpClient.Do(req, &users, nil)
	if err != nil {
		return nil, errorsutils.NewErrorWithExitCode(fmt.Errorf("failed to get user response: %w", err), statusCode)
	}
	return users, nil
}