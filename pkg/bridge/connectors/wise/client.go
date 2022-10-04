package wise

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const apiEndpoint = "https://api.wise.com"

type apiTransport struct {
	ApiKey string
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.ApiKey))
	return http.DefaultTransport.RoundTrip(req)
}

type client struct {
	httpClient *http.Client
}

type profile struct {
	ID   uint64 `json:"id"`
	Type string `json:"type"`
}

type transfer struct {
	ID                    uint64  `json:"id"`
	Reference             string  `json:"reference"`
	Status                string  `json:"status"`
	SourceAccount         uint64  `json:"sourceAccount"`
	SourceCurrency        string  `json:"sourceCurrency"`
	SourceValue           float64 `json:"sourceValue"`
	TargetAccount         uint64  `json:"targetAccount"`
	TargetCurrency        string  `json:"targetCurrency"`
	TargetValue           float64 `json:"targetValue"`
	Business              uint64  `json:"business"`
	Created               string  `json:"created"`
	CustomerTransactionId string  `json:"customerTransactionId"`
	Details               struct {
		Reference string `json:"reference"`
	} `json:"details"`
	Rate float64 `json:"rate"`
	User uint64  `json:"user"`
}

func (w *client) endpoint(path string) string {
	return fmt.Sprintf("%s/%s", apiEndpoint, path)
}

func (w *client) getProfiles() ([]profile, error) {
	var profiles []profile

	res, err := w.httpClient.Get(w.endpoint("v1/profiles"))
	if err != nil {
		return profiles, err
	}

	b, _ := io.ReadAll(res.Body)

	err = json.Unmarshal(b, &profiles)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal profiles: %w", err)
	}

	return profiles, nil
}

func (w *client) getTransfers(profile *profile) ([]transfer, error) {
	var transfers []transfer

	limit := 10
	offset := 0

	for {
		var ts []transfer

		req, err := http.NewRequest(http.MethodGet, w.endpoint("v1/transfers"), nil)
		if err != nil {
			return transfers, err
		}

		q := req.URL.Query()
		q.Add("limit", fmt.Sprintf("%d", limit))
		q.Add("profile", fmt.Sprintf("%d", profile.ID))
		q.Add("offset", fmt.Sprintf("%d", offset))
		req.URL.RawQuery = q.Encode()

		res, err := w.httpClient.Do(req)
		if err != nil {
			return transfers, err
		}

		b, _ := io.ReadAll(res.Body)
		err = json.Unmarshal(b, &ts)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal transfers: %w", err)
		}

		transfers = append(transfers, ts...)

		if len(ts) < limit {
			break
		}

		offset += limit
	}

	return transfers, nil
}

func newClient(apiKey string) *client {
	httpClient := &http.Client{
		Transport: &apiTransport{
			ApiKey: apiKey,
		},
	}

	return &client{
		httpClient: httpClient,
	}
}
