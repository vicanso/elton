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
	"net/http"
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

const (
	defaultRealm         = "basic auth tips"
	errBasicAuthCategory = "cod-basic-auth"
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
	errUnauthorized = getBasicAuthError("unAuthorized", http.StatusUnauthorized)
)

func getBasicAuthError(message string, statusCode int) *hes.Error {
	return &hes.Error{
		StatusCode: statusCode,
		Message:    message,
		Category:   errBasicAuthCategory,
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
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skiper(c) {
			return c.Next()
		}
		auth := c.Request.Header.Get(cod.HeaderAuthorization)
		if len(auth) < basicLen+1 ||
			strings.ToLower(auth[:basicLen]) != basic {
			c.SetHeader(cod.HeaderWWWAuthenticate, wwwAuthenticate)
			err = errUnauthorized
			return
		}

		v, e := base64.StdEncoding.DecodeString(auth[basicLen+1:])
		if e != nil {
			err = getBasicAuthError(e.Error(), http.StatusBadRequest)
			return err
		}

		arr := strings.Split(string(v), ":")
		valid, e := config.Validate(arr[0], arr[1], c)

		if e != nil {
			err, ok := e.(*hes.Error)
			if !ok {
				err = getBasicAuthError(e.Error(), http.StatusBadRequest)
			}
			return err
		}
		if !valid {
			c.SetHeader(cod.HeaderWWWAuthenticate, wwwAuthenticate)
			err = errUnauthorized
			return
		}
		return c.Next()
	}
}
