package models

type PSPWebhookConfig struct {
	Name    string `json:"name"`
	URLPath string `json:"urlPath"`
}

type WebhookConfig struct {
	Name        string      `json:"name"`
	ConnectorID ConnectorID `json:"connectorID"`
	URLPath     string      `json:"urlPath"`
}

type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type PSPWebhook struct {
	BasicAuth *BasicAuth `json:"basicAuth"`

	QueryValues map[string][]string `json:"queryValues"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"payload"`
}

type Webhook struct {
	ID          string              `json:"id"`
	ConnectorID ConnectorID         `json:"connectorID"`
	BasicAuth   *BasicAuth          `json:"basicAuth"`
	QueryValues map[string][]string `json:"queryValues"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"payload"`
}
