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
	"github.com/vicanso/hes"
)

func TestNoVildatePanic(t *testing.T) {
	assert := assert.New(t)
	defer func() {
		r := recover()
		assert.NotNil(r)
		assert.Equal(r.(error), ErrBasicAuthRequireValidateFunction)
	}()

	NewBasicAuth(BasicAuthConfig{})
}

func TestBasicAuth(t *testing.T) {
	m := NewBasicAuth(BasicAuthConfig{
		Validate: func(account, pwd string, c *elton.Context) (bool, error) {
			if account == "tree.xie" && pwd == "password" {
				return true, nil
			}
			if account == "n" {
				return false, hes.New("account is invalid")
			}
			return false, nil
		},
	})
	req := httptest.NewRequest("GET", "https://aslant.site/", nil)

	t.Run("skip", func(t *testing.T) {
		assert := assert.New(t)
		done := false
		mSkip := NewBasicAuth(BasicAuthConfig{
			Validate: func(account, pwd string, c *elton.Context) (bool, error) {
				return false, nil
			},
			Skipper: func(c *elton.Context) bool {
				return true
			},
		})
		e := elton.New()
		e.Use(mSkip)
		e.GET("/", func(c *elton.Context) error {
			done = true
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.True(done)
	})

	t.Run("no auth header", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(resp.Code, http.StatusUnauthorized)
		assert.Equal(resp.Header().Get(elton.HeaderWWWAuthenticate), `basic realm="basic auth tips"`)
	})

	t.Run("auth validate fail", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		e.GET("/", func(c *elton.Context) error {
			return nil
		})
		req.Header.Set(elton.HeaderAuthorization, "basic YTpi")
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(resp.Code, http.StatusUnauthorized)
		assert.Equal(resp.Body.String(), "category=elton-basic-auth, message=unAuthorized")

		req.Header.Set(elton.HeaderAuthorization, "basic bjph")
		resp = httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(resp.Code, http.StatusBadRequest)
		assert.Equal(resp.Body.String(), "message=account is invalid")
	})

	t.Run("validate error", func(t *testing.T) {
		assert := assert.New(t)
		mValidateFail := NewBasicAuth(BasicAuthConfig{
			Validate: func(account, pwd string, c *elton.Context) (bool, error) {
				return false, errors.New("abcd")
			},
		})
		e := elton.New()
		e.Use(mValidateFail)
		e.GET("/", func(c *elton.Context) error {
			return nil
		})
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.Equal(resp.Code, http.StatusBadRequest)
		assert.Equal(resp.Body.String(), "category=elton-basic-auth, message=abcd")
	})

	t.Run("auth success", func(t *testing.T) {
		assert := assert.New(t)
		e := elton.New()
		e.Use(m)
		done := false
		e.GET("/", func(c *elton.Context) error {
			done = true
			return nil
		})
		req.Header.Set(elton.HeaderAuthorization, "basic dHJlZS54aWU6cGFzc3dvcmQ=")
		resp := httptest.NewRecorder()
		e.ServeHTTP(resp, req)
		assert.True(done)
	})
}
