package rest

import (
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
	if err == nil {
		return false
	}
	if errWithStatusCode, ok := err.(ErrorWithStatusCode); ok {
		return errWithStatusCode.StatusCode == code
	}
	return false
}
