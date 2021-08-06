// MIT License

// Copyright (c) 2020 Tree Xie

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package middleware

import (
	"errors"
	"net/http"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

const (
	defaultBasicAuthRealm = "basic auth"
	// ErrBasicAuthCategory basic auth error category
	ErrBasicAuthCategory = "elton-basic-auth"
)

type (
	// BasicAuthValidate validate function
	BasicAuthValidate func(username string, password string, c *elton.Context) (bool, error)
	// BasicAuthConfig basic auth config
	BasicAuthConfig struct {
		Realm    string
		Validate BasicAuthValidate
		Skipper  elton.Skipper
	}
)

var (
	// ErrBasicAuthUnauthorized unauthorized err
	ErrBasicAuthUnauthorized = getBasicAuthError(errors.New("unAuthorized"), http.StatusUnauthorized)
	// ErrBasicAuthRequireValidateFunction require validate function
	ErrBasicAuthRequireValidateFunction = errors.New("require validate function")
)

func getBasicAuthError(err error, statusCode int) *hes.Error {
	he := hes.Wrap(err)
	he.StatusCode = statusCode
	he.Category = ErrBasicAuthCategory
	return he
}

// NewDefaultBasicAuth returns a new basic auth middleware,
// it will check the account and password, it will returns an error if check failed
func NewDefaultBasicAuth(account, password string) elton.Handler {
	return NewBasicAuth(BasicAuthConfig{
		Validate: func(acc, pwd string, c *elton.Context) (bool, error) {
			if acc == account && pwd == password {
				return true, nil
			}
			return false, nil
		},
	})
}

// NewBasicAuth create a basic auth middleware, it will throw an error if the the validate function is nil.
func NewBasicAuth(config BasicAuthConfig) elton.Handler {
	if config.Validate == nil {
		panic(ErrBasicAuthRequireValidateFunction)
	}
	basic := "basic"
	realm := defaultBasicAuthRealm
	if config.Realm != "" {
		realm = config.Realm
	}
	wwwAuthenticate := basic + ` realm="` + realm + `"`
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) error {
		if skipper(c) || c.Request.Method == http.MethodOptions {
			return c.Next()
		}

		user, password, hasAuth := c.Request.BasicAuth()
		// 如果请求头无认证头，则返回出错
		if !hasAuth {
			c.SetHeader(elton.HeaderWWWAuthenticate, wwwAuthenticate)
			return ErrBasicAuthUnauthorized
		}

		valid, e := config.Validate(user, password, c)

		// 如果返回出错，则输出出错信息
		if e != nil {
			err, ok := e.(*hes.Error)
			if !ok {
				err = getBasicAuthError(e, http.StatusBadRequest)
			}
			return err
		}

		// 如果校验失败，设置认证头，客户重新输入
		if !valid {
			c.SetHeader(elton.HeaderWWWAuthenticate, wwwAuthenticate)
			return ErrBasicAuthUnauthorized
		}
		return c.Next()
	}
}
