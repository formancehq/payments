package wise

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/numary/go-libs/sharedlogging"
)

const (
	apiEndpoint = "https://api.wise.com"
)

type apiTransport struct {
	ApiKey string
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.ApiKey))
	return http.DefaultTransport.RoundTrip(req)
}

type WiseClient struct {
	httpClient *http.Client
}

type Profile struct {
	Id   uint64 `json:"id"`
	Type string `json:"type"`
}

type Transfer struct {
	ID                    uint64  `json:"id"`
	Reference             string  `json:"reference"`
	Status                string  `json:"status"`
	SourceAccount         uint64  `json:"sourceAccount"`
	SourceCurrency        string  `json:"sourceCurrency"`
	SourceValue           float64 `json:"sourceValue"`
	TargetAccount         uint64  `json:"targetAccount"`
	TargetCurrency        string  `json:"targetCurrency"`
	TargetValue           float64 `json:"targetValue"`
	Business              string  `json:"business"`
	Created               string  `json:"created"`
	CustomerTransactionId string  `json:"customerTransactionId"`
	Details               struct {
		Reference string `json:"reference"`
	} `json:"details"`
	Rate float64 `json:"rate"`
	User uint64  `json:"user"`
}

type BalanceAccount struct {
	ID           uint64 `json:"id"`
	Type         string `json:"type"`
	Currency     string `json:"currency"`
	CreationTime string `json:"creationTime"`
	Name         string `json:"name"`
	Amount       struct {
		Value    float64 `json:"value"`
		Currency string  `json:"currency"`
	} `json:"amount"`
}

func (w *WiseClient) Endpoint(path string) string {
	return fmt.Sprintf("%s/%s", apiEndpoint, path)
}

func (w *WiseClient) GetProfiles() ([]Profile, error) {
	profiles := []Profile{}

	res, err := w.httpClient.Get(w.Endpoint("v1/profiles"))
	if err != nil {
		return profiles, err
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &profiles); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (w *WiseClient) GetTransfers(ctx context.Context, profile *Profile) ([]Transfer, error) {
	transfers := []Transfer{}

	limit := 10
	offset := 0

	for {
		ts := []Transfer{}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.Endpoint("v1/transfers"), nil)
		if err != nil {
			return transfers, err
		}

		q := req.URL.Query()
		q.Add("limit", fmt.Sprintf("%d", limit))
		q.Add("profile", fmt.Sprintf("%d", profile.Id))
		q.Add("offset", fmt.Sprintf("%d", offset))
		req.URL.RawQuery = q.Encode()

		res, err := w.httpClient.Do(req)
		if err != nil {
			return transfers, err
		}

		if err := json.NewDecoder(res.Body).Decode(&ts); err != nil {
			sharedlogging.GetLogger(ctx).Errorf("unmarshalling response: %s", err)
			continue
		}

		transfers = append(transfers, ts...)

		if len(ts) < limit {
			break
		}

		offset += limit
	}

	return transfers, nil
}

func NewClient(apiKey string) *WiseClient {
	httpClient := &http.Client{
		Transport: &apiTransport{
			ApiKey: apiKey,
		},
	}

	return &WiseClient{
		httpClient: httpClient,
	}
}
