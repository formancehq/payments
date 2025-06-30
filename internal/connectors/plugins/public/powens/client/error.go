package client

import (
	"fmt"
)

type powensError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Message     string `json:"message"`
	RequestID   string `json:"requestId"`
}

func (pe *powensError) Error() error {
	return fmt.Errorf("%s: %s", pe.Code, pe.Description)
}
