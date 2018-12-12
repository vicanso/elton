package middleware

import (
	"fmt"
	"net/http"

	"github.com/vicanso/cod"
)

// NewRecover new recover
func NewRecover() cod.Handler {
	return func(c *cod.Context) error {
		defer func() {
			// 此recover只是示例，在实际使用中，
			// 需要针对实际需求调整，如对于每个recover增加邮件通知等
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
