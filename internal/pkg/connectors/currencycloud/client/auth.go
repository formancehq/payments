package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) authenticate() (string, error) {
	form := make(url.Values)

	form.Add("login_id", c.loginID)
	form.Add("api_key", c.apiKey)

	resp, err := c.httpClient.PostForm(c.buildEndpoint("v2/authenticate/api"), form)
	if err != nil {
		return "", fmt.Errorf("failed to do get request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	//nolint:tagliatelle // allow for client code
	type response struct {
		AuthToken string `json:"auth_token"`
	}

	var res response

	if err = json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("failed to decode response body: %w", err)
	}

	return res.AuthToken, nil
}
