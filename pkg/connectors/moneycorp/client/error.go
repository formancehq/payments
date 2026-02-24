package client

import (
	"fmt"
)

type moneycorpErrors struct {
	Errors []*moneycorpError `json:"errors"`
}

func (mes *moneycorpErrors) Error() error {
	return toError(0, *mes).Error()
}

type moneycorpError struct {
	Status int    `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func (me *moneycorpError) Error() error {
	var err error
	if me.Detail == "" {
		err = fmt.Errorf("unexpected status code: %d", me.Status)
	} else {
		err = fmt.Errorf("%d: %s", me.Status, me.Detail)
	}

	return err
}

func toError(statusCode int, ces moneycorpErrors) *moneycorpError {
	if len(ces.Errors) == 0 {
		return &moneycorpError{
			Status: statusCode,
		}
	}

	return &moneycorpError{
		Status: ces.Errors[0].Status,
		Code:   ces.Errors[0].Code,
		Title:  ces.Errors[0].Title,
		Detail: ces.Errors[0].Detail,
	}
}
