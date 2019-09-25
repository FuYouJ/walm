package error

import "github.com/pkg/errors"

type NotFoundError struct {}

func (err NotFoundError) Error() string {
	return "not found error"
}

func IsNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(NotFoundError)
	return ok
}