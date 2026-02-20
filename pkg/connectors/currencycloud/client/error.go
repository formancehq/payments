package client

import (
	"fmt"
)

type currencyCloudError struct {
	StatusCode    int                        `json:"status_code"`
	ErrorCode     string                     `json:"error_code"`
	ErrorMessages map[string][]*errorMessage `json:"error_messages"`
}

type errorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (ce *currencyCloudError) Error() error {
	var errorMessage string
	if len(ce.ErrorMessages) > 0 {
		for _, message := range ce.ErrorMessages {
			if len(message) > 0 {
				errorMessage = message[0].Message
				break
			}
		}
	}

	if errorMessage == "" {
		return fmt.Errorf("unexpected status code: %d", ce.StatusCode)
	}

	return fmt.Errorf("%s: %s", ce.ErrorCode, errorMessage)
}
