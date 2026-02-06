package client

import (
	"errors"
	"fmt"
)

type modulrErrors []modulrError

type modulrError struct {
	StatusCode    int    `json:"-"`
	Field         string `json:"field"`
	Code          string `json:"code"`
	Message       string `json:"message"`
	ErrorCode     string `json:"errorCode"`
	SourceService string `json:"sourceService"`
}

func (mes modulrErrors) Error() error {
	if len(mes) == 0 {
		return errors.New("unexpected error")
	}

	me := mes[0]

	var err error
	if me.Message == "" {
		err = fmt.Errorf("unexpected status code: %d", me.StatusCode)
	} else {
		err = fmt.Errorf("%d: %s", me.StatusCode, me.Message)
	}

	return err
}
