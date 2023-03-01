package client

import "errors"

var (
	ErrCreateRequest     = errors.New("error creating a request")
	ErrInvalidStatusCode = errors.New("invalid status code")
	ErrDoRequest         = errors.New("error performing request to reddit api")
)
