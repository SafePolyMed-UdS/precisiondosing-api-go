package queryerr

import (
	"encoding/json"
	"fmt"
)

type Error struct {
	StatusCode int
	Message    json.RawMessage
	Internal   error
}

func NewHTTTP(statusCode int, message json.RawMessage) *Error {
	return &Error{
		StatusCode: statusCode,
		Message:    message,
		Internal:   nil,
	}
}

func NewInternal(internal error) *Error {
	return &Error{
		StatusCode: 0,
		Message:    nil,
		Internal:   internal,
	}
}

func (e *Error) IsHTTPError() bool {
	return e.StatusCode > 0
}

func (e *Error) IsInternalError() bool {
	return e.StatusCode <= 0
}

func (e *Error) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("internal query error: %s", e.Internal.Error())
	} else if e.StatusCode != 0 {
		return fmt.Sprintf("query error with status code %d: %s", e.StatusCode, string(e.Message))
	}
	return "Query Error: Unknown error"
}

func (e *Error) Unwrap() error {
	return e.Internal
}
