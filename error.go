package cod

import (
	"fmt"
)

const (
	// ErrCategoryCod cod category
	ErrCategoryCod = "cod"
)

var (
	// ErrInvalidRedirect invalid redirect error
	ErrInvalidRedirect = NewError(400, "invalid redirect", ErrCategoryCod)
	// ErrInvalidResponse invalid response(body an status is nil)
	ErrInvalidResponse = NewError(500, "invalid response", ErrCategoryCod)
)

type (
	// HTTPError http error
	HTTPError struct {
		StatusCode int                    `json:"status_code,omitempty"`
		Code       string                 `json:"code,omitempty"`
		Category   string                 `json:"category,omitempty"`
		Message    string                 `json:"message,omitempty"`
		Exception  bool                   `json:"exception,omitempty"`
		Extra      map[string]interface{} `json:"extra,omitempty"`
	}
)

// Error error interface
func (e *HTTPError) Error() string {
	str := fmt.Sprintf("status=%d, message=%s", e.StatusCode, e.Message)
	if e.Category == "" {
		return str
	}
	return fmt.Sprintf("category=%s, %s", e.Category, str)
}

// NewError create an error
func NewError(statusCode int, message, category string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Category:   category,
	}
}
