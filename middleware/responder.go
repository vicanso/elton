package middleware

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/vicanso/cod"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

type (
	// ResponderConfig response config
	ResponderConfig struct {
	}
)

// NewResponder create a responder
func NewResponder(config ResponderConfig) cod.Handle {
	return func(c *cod.Context) error {
		e := c.Next()
		var err *cod.HTTPError
		if e != nil {
			// 如果出错，尝试转换为HTTPError
			he, ok := e.(*cod.HTTPError)
			if !ok {
				he = &cod.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: e.Error(),
				}
			}
			err = he
		} else if c.Status == 0 && c.Body == nil {
			// 如果status与body都为空，则为非法响应
			err = cod.ErrInvalidResponse
		}

		resp := c.Response
		respHeader := resp.Header()
		ct := cod.HeaderContentType

		if err != nil {

			c.Status = err.Code
			c.Body, _ = json.Marshal(err)
			respHeader.Set(ct, cod.MIMEApplicationJSON)
		}

		hadContentType := false
		// 判断是否已设置响应头的Content-Type
		if respHeader.Get(ct) != "" {
			hadContentType = true
		}

		status := c.Status
		if status == 0 {
			status = http.StatusOK
		}

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
					status = http.StatusInternalServerError
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
		c.Response.WriteHeader(status)
		_, responseErr := resp.Write(body)

		if responseErr != nil {
			// TODO 如何触发实例error
			c.Cod().EmitError(c, responseErr)
		}

		return nil
	}
}
