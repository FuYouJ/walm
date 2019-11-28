package error

import "github.com/pkg/errors"

type NotFoundError struct {
	Message string
}

func (err NotFoundError) Error() string {
	if err.Message == "" {
		return "not found error"
	}
	return err.Message

}

func IsNotFoundError(err error) bool {
	_, ok := errors.Cause(err).(NotFoundError)
	return ok
}
