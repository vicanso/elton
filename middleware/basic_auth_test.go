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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestNoVildatePanic(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.NotNil(r)
		assert.Equal(ErrBasicAuthRequireValidateFunction, r.(error))
	}()

	NewBasicAuth(BasicAuthConfig{})
}

func TestBasicAuthSkip(t *testing.T) {
	assert := assert.New(t)
	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}
	defaultAuth := NewBasicAuth(BasicAuthConfig{
		Validate: func(acccount, pwd string, c *elton.Context) (bool, error) {
			return true, nil
		},
	})
	tests := []struct {
		newContext func() *elton.Context
		err        error
		fn         elton.Handler
		headerAuth string
	}{
		// committed: true
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), nil)
				c.Committed = true
				c.Next = next
				return c
			},
			err: skipErr,
			fn:  defaultAuth,
		},
		// options method
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("OPTIONS", "/", nil)
				c := elton.NewContext(httptest.NewRecorder(), req)
				c.Next = next
				return c
			},
			err: skipErr,
			fn:  defaultAuth,
		},
		// not set auth header
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				return c
			},
			err:        ErrBasicAuthUnauthorized,
			fn:         defaultAuth,
			headerAuth: `basic realm="basic auth"`,
		},
		// validate return error
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				c.Request.SetBasicAuth("account", "password")
				return c
			},
			err: getBasicAuthError(errors.New("custom error"), http.StatusBadRequest),
			fn: NewBasicAuth(BasicAuthConfig{
				Validate: func(account, password string, _ *elton.Context) (bool, error) {
					return false, errors.New("custom error")
				},
			}),
		},
		// validate fail
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				c.Request.SetBasicAuth("account", "pass")
				return c
			},
			err:        ErrBasicAuthUnauthorized,
			headerAuth: `basic realm="custom realm"`,
			fn: NewBasicAuth(BasicAuthConfig{
				Validate: func(account, password string, _ *elton.Context) (bool, error) {
					return false, nil
				},
				Realm: "custom realm",
			}),
		},
		// success
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
				c.Request.SetBasicAuth("account", "password")
				c.Next = next
				return c
			},
			fn:  NewDefaultBasicAuth("account", "password"),
			err: skipErr,
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := tt.fn(c)
		assert.Equal(tt.err, err)
		assert.Equal(tt.headerAuth, c.GetHeader(elton.HeaderWWWAuthenticate))
	}
}
