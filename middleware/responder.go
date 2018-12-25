package middleware

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/vicanso/cod"
	"github.com/vicanso/errors"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type (
	// ResponderConfig response config
	ResponderConfig struct {
		Skipper Skipper
	}
)

// NewResponder create a responder
func NewResponder(config ResponderConfig) cod.Handler {
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) error {
		if skiper(c) {
			return c.Next()
		}
		e := c.Next()
		var err *errors.HTTPError
		if e != nil {
			// 如果出错，尝试转换为HTTPError
			he, ok := e.(*errors.HTTPError)
			if !ok {
				he = &errors.HTTPError{
					StatusCode: http.StatusInternalServerError,
					Message:    e.Error(),
				}
			}
			err = he
		} else if c.StatusCode == 0 && c.Body == nil {
			// 如果status code与body都为空，则为非法响应
			err = cod.ErrInvalidResponse
		}

		respHeader := c.Headers
		ct := cod.HeaderContentType

		// 从出错中获取响应数据，响应状态码
		if err != nil {
			c.StatusCode = err.StatusCode
			c.Body, _ = json.Marshal(err)
			respHeader.Set(ct, cod.MIMEApplicationJSON)
		}

		hadContentType := false
		// 判断是否已设置响应头的Content-Type
		if respHeader.Get(ct) != "" {
			hadContentType = true
		}

		if c.StatusCode == 0 {
			c.StatusCode = http.StatusOK
		}
		statusCode := c.StatusCode

		var body []byte
		if c.Body != nil {
			switch c.Body.(type) {
			case string:
				if !hadContentType {
					respHeader.Set(ct, cod.MIMETextPlain)
					body = []byte(c.Body.(string))
				}
			case []byte:
				if !hadContentType {
					respHeader.Set(ct, cod.MIMEBinary)
				}
				body = c.Body.([]byte)
			default:
				// 转换为json
				buf, err := json.Marshal(c.Body)
				if err != nil {
					statusCode = http.StatusInternalServerError
					body = []byte(err.Error())
				} else {
					if !hadContentType {
						respHeader.Set(ct, cod.MIMEApplicationJSON)
					}
					body = buf
				}
			}
		}
		c.BodyBytes = body
		c.StatusCode = statusCode

		return nil
	}
}
