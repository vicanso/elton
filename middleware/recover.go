package middleware

import (
	"fmt"
	"net/http"

	"github.com/vicanso/cod"
)

// NewRecover new recover
func NewRecover() cod.Handle {
	return func(c *cod.Context) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				resp := c.Response
				resp.WriteHeader(http.StatusInternalServerError)
				resp.Write([]byte(err.Error()))
			}
		}()
		return c.Next()
	}
}
