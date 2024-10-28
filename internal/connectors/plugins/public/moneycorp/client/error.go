package client

import (
	"fmt"
)

type moneycorpErrors struct {
	Errors []*moneycorpError `json:"errors"`
}

type moneycorpError struct {
	StatusCode int    `json:"-"`
	Code       string `json:"code"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
}

func (me *moneycorpError) Error() error {
	var err error
	if me.Detail == "" {
		err = fmt.Errorf("unexpected status code: %d", me.StatusCode)
	} else {
		err = fmt.Errorf("%d: %s", me.StatusCode, me.Detail)
	}

	return err
}

func toError(statusCode int, ces moneycorpErrors) *moneycorpError {
	if len(ces.Errors) == 0 {
		return &moneycorpError{
			StatusCode: statusCode,
		}
	}

	return &moneycorpError{
		StatusCode: statusCode,
		Code:       ces.Errors[0].Code,
		Title:      ces.Errors[0].Title,
		Detail:     ces.Errors[0].Detail,
	}
}
