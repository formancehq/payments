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
	Detail string `json:"detail"`
}

func (errorResponse qontoErrors) Error() error {
	var err error
	if len(errorResponse.Errors) == 0 {
		err = fmt.Errorf("statusCode=%d, errorMessage=\"unexpected error\"", errorResponse.StatusCode)
		return err
	}

	if len(errorResponse.Errors) == 1 {
		me := errorResponse.Errors[0]
		return fmt.Errorf(
			"statusCode=%d, errorCode=\"%s\", errorMessage=\"%s\"",
			errorResponse.StatusCode,
			me.Code,
			me.Detail,
		)
	}

	errMsg := fmt.Sprintf("multiple errors (statusCode=%d):", errorResponse.StatusCode)
	for _, e := range errorResponse.Errors {
		errMsg += fmt.Sprintf(" [errorCode=\"%s\", errorMessage=\"%s\"]", e.Code, e.Detail)
	}
	err = fmt.Errorf(errMsg)
	return err
}
