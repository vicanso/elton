package middleware

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/vicanso/cod"
)

func TestErrorHandler(t *testing.T) {
	fn := NewDefaultErrorHandler()
	req := httptest.NewRequest("GET", "/users/me", nil)
	resp := httptest.NewRecorder()
	c := cod.NewContext(resp, req)
	c.Next = func() error {
		return errors.New("abcd")
	}
	err := fn(c)
	if err != nil {
		t.Fatalf("error handler fail, %v", err)
	}
	ct := c.GetHeader(cod.HeaderContentType)
	if c.BodyBuffer.String() != `{"statusCode":500,"category":"cod-error-handler","message":"abcd"}` ||
		ct != "application/json; charset=UTF-8" {
		t.Fatalf("error handler fail")
	}
}
