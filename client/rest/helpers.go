package rest

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorWithStatusCode struct {
	StatusCode int
	Err        error
}

func (e ErrorWithStatusCode) Error() string {
	return fmt.Sprintf("%s (%d): %s", http.StatusText(e.StatusCode), e.StatusCode, e.Err.Error())
}

func HasStatusCode(err error, code int) bool {
	var errWithStatusCode ErrorWithStatusCode
	return errors.As(err, &errWithStatusCode) && errWithStatusCode.StatusCode == code
}
