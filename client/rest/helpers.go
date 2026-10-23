package rest

import (
	"errors"
	"fmt"
	"net/http"
)

type ErrorWithStatusCode struct {
	StatusCode int
	// Type is the machine-readable error type from the response body, if any.
	Type string
	Err  error
}

func (e ErrorWithStatusCode) Error() string {
	return fmt.Sprintf("%s (%d): %s", http.StatusText(e.StatusCode), e.StatusCode, e.Err.Error())
}

func HasStatusCode(err error, code int) bool {
	var errWithStatusCode ErrorWithStatusCode
	return errors.As(err, &errWithStatusCode) && errWithStatusCode.StatusCode == code
}

// HasErrorType reports whether err carries the given response error type.
func HasErrorType(err error, errorType string) bool {
	var errWithStatusCode ErrorWithStatusCode
	return errors.As(err, &errWithStatusCode) && errWithStatusCode.Type == errorType
}
