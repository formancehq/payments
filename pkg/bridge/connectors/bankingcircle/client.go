package bankingcircle

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/numary/go-libs/sharedlogging"
)

type client struct {
	httpClient *http.Client

	username string
	password string

	endpoint              string
	authorizationEndpoint string

	logger sharedlogging.Logger

	accessToken          string
	accessTokenExpiresAt time.Time
}

func newClient(username, password, endpoint, authorizationEndpoint string, logger sharedlogging.Logger) (*client, error) {
	c := &client{
		httpClient: &http.Client{Timeout: 10 * time.Second},

		username:              username,
		password:              password,
		endpoint:              endpoint,
		authorizationEndpoint: authorizationEndpoint,

		logger: logger,
	}

	if err := c.login(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *client) login() error {
	req, err := http.NewRequest(http.MethodGet, c.authorizationEndpoint+"/api/v1/authorizations/authorize", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			c.logger.Error(err)
		}
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response body: %w", err)
	}

	type response struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	fmt.Println(resp.Status)
	fmt.Println(string(responseBody))
	var res response

	if err = json.Unmarshal(responseBody, &res); err != nil {
		return fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	c.accessToken = res.AccessToken
	c.accessTokenExpiresAt = time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)

	return nil
}

func (c *client) ensureAccessTokenIsValid() error {
	if c.accessTokenExpiresAt.After(time.Now()) {
		return nil
	}

	return c.login()
}

type payment struct {
	PaymentID            string      `json:"paymentId"`
	TransactionReference string      `json:"transactionReference"`
	ConcurrencyToken     string      `json:"concurrencyToken"`
	Classification       string      `json:"classification"`
	Status               string      `json:"status"`
	Errors               interface{} `json:"errors"`
	LastChangedTimestamp time.Time   `json:"lastChangedTimestamp"`
	DebtorInformation    struct {
		PaymentBulkID interface{} `json:"paymentBulkId"`
		AccountID     string      `json:"accountId"`
		Account       struct {
			Account              string `json:"account"`
			FinancialInstitution string `json:"financialInstitution"`
			Country              string `json:"country"`
		} `json:"account"`
		VibanID interface{} `json:"vibanId"`
		Viban   struct {
			Account              string `json:"account"`
			FinancialInstitution string `json:"financialInstitution"`
			Country              string `json:"country"`
		} `json:"viban"`
		InstructedDate interface{} `json:"instructedDate"`
		DebitAmount    struct {
			Currency string  `json:"currency"`
			Amount   float64 `json:"amount"`
		} `json:"debitAmount"`
		DebitValueDate time.Time   `json:"debitValueDate"`
		FxRate         interface{} `json:"fxRate"`
		Instruction    interface{} `json:"instruction"`
	} `json:"debtorInformation"`
	Transfer struct {
		DebtorAccount interface{} `json:"debtorAccount"`
		DebtorName    interface{} `json:"debtorName"`
		DebtorAddress interface{} `json:"debtorAddress"`
		Amount        struct {
			Currency string  `json:"currency"`
			Amount   float64 `json:"amount"`
		} `json:"amount"`
		ValueDate             interface{} `json:"valueDate"`
		ChargeBearer          interface{} `json:"chargeBearer"`
		RemittanceInformation interface{} `json:"remittanceInformation"`
		CreditorAccount       interface{} `json:"creditorAccount"`
		CreditorName          interface{} `json:"creditorName"`
		CreditorAddress       interface{} `json:"creditorAddress"`
	} `json:"transfer"`
	CreditorInformation struct {
		AccountID string `json:"accountId"`
		Account   struct {
			Account              string `json:"account"`
			FinancialInstitution string `json:"financialInstitution"`
			Country              string `json:"country"`
		} `json:"account"`
		VibanID interface{} `json:"vibanId"`
		Viban   struct {
			Account              string `json:"account"`
			FinancialInstitution string `json:"financialInstitution"`
			Country              string `json:"country"`
		} `json:"viban"`
		CreditAmount struct {
			Currency string  `json:"currency"`
			Amount   float64 `json:"amount"`
		} `json:"creditAmount"`
		CreditValueDate time.Time   `json:"creditValueDate"`
		FxRate          interface{} `json:"fxRate"`
	} `json:"creditorInformation"`
}

func (c *client) getAllPayments() ([]*payment, error) {
	var payments []*payment

	for page := 0; ; page++ {
		pagedPayments, err := c.getPayments(page)
		if err != nil {
			return nil, err
		}

		if len(pagedPayments) == 0 {
			break
		}

		payments = append(payments, pagedPayments...)
	}

	return payments, nil
}

func (c *client) getPayments(page int) ([]*payment, error) {
	if err := c.ensureAccessTokenIsValid(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, c.endpoint+"/api/v1/payments/singles", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	q := req.URL.Query()
	q.Add("PageSize", "5000")
	q.Add("PageNumber", fmt.Sprint(page))

	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			c.logger.Error(err)
		}
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read login response body: %w", err)
	}

	type response struct {
		Result   []*payment `json:"result"`
		PageInfo struct {
			CurrentPage int `json:"currentPage"`
			PageSize    int `json:"pageSize"`
		} `json:"pageInfo"`
	}

	var res response

	if err = json.Unmarshal(responseBody, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal login response: %w", err)
	}

	return res.Result, nil
}
