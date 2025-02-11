package client

import "fmt"

type increaseError struct {
	Status int    `json:"status"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Errors []struct {
		Field   string `json:"field"`
		Message string `json:"message"`
	} `json:"errors"`
}

func (ie *increaseError) Error() error {
	var err error
	if ie.Detail == "" {
		err = fmt.Errorf("unexpected status code: %d", ie.Status)
	} else {
		err = fmt.Errorf("%d: %s", ie.Status, ie.Detail)
	}

	return err
}
