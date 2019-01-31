// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package middleware

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

const (
	defaultRealm = "basic auth tips"
)

type (
	// Validate validate function
	Validate func(string, string, *cod.Context) (bool, error)
	// BasicAuthConfig basic auth config
	BasicAuthConfig struct {
		Realm    string
		Validate Validate
		Skipper  Skipper
	}
)

var (
	// errUnauthorized unauthorized err
	errUnauthorized = getBasicAuthError(errors.New("unAuthorized"), http.StatusUnauthorized)
)

func getBasicAuthError(err error, statusCode int) *hes.Error {
	return &hes.Error{
		StatusCode: statusCode,
		Message:    err.Error(),
		Category:   ErrCategoryBasicAuth,
		Err:        err,
	}
}

// NewBasicAuth new basic auth
func NewBasicAuth(config BasicAuthConfig) cod.Handler {
	if config.Validate == nil {
		panic("require validate function")
	}
	basic := "basic"
	basicLen := len(basic)
	realm := defaultRealm
	if config.Realm != "" {
		realm = config.Realm
	}
	wwwAuthenticate := basic + " realm=" + realm
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		auth := c.Request.Header.Get(cod.HeaderAuthorization)
		// 如果请求头无认证头，则返回出错
		if len(auth) < basicLen+1 ||
			strings.ToLower(auth[:basicLen]) != basic {
			c.SetHeader(cod.HeaderWWWAuthenticate, wwwAuthenticate)
			err = errUnauthorized
			return
		}

		v, e := base64.StdEncoding.DecodeString(auth[basicLen+1:])
		// base64 decode 失败
		if e != nil {
			err = getBasicAuthError(e, http.StatusBadRequest)
			return
		}

		arr := strings.Split(string(v), ":")
		// 调用校验函数
		valid, e := config.Validate(arr[0], arr[1], c)

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
			c.SetHeader(cod.HeaderWWWAuthenticate, wwwAuthenticate)
			err = errUnauthorized
			return
		}
		return c.Next()
	}
}
