package client

import (
	"fmt"
)

type qontoErrors struct {
	StatusCode int `json:"-"`
	Errors     []qontoError
}

type qontoError struct {
	Code   string `json:"code"`
	Detail string `json:"Detail"`
}

func (errorResponse qontoErrors) Error() error {
	var err error
	if len(errorResponse.Errors) == 0 {
		err = fmt.Errorf("unexpected error, status code: %d", errorResponse.StatusCode)
		return err
	}

	me := errorResponse.Errors[0]

	err = fmt.Errorf("%s: %s", me.Code, me.Detail)

	return err
}
