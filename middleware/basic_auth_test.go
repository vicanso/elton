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
	tests := []struct {
		newContext func() *elton.Context
	}{
		// commited: true
		{
			newContext: func() *elton.Context {
				c := elton.NewContext(nil, nil)
				c.Committed = true
				c.Next = next
				return c
			},
		},
		// options method
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("OPTIONS", "/", nil)
				c := elton.NewContext(nil, req)
				c.Next = next
				return c
			},
		},
	}
	fn := NewBasicAuth(BasicAuthConfig{
		Validate: func(acccount, pwd string, c *elton.Context) (bool, error) {
			return true, nil
		},
	})
	for _, tt := range tests {
		err := fn(tt.newContext())
		assert.Equal(skipErr, err)
	}
}

func TestBasicAuthNotSetAuthHeader(t *testing.T) {
	assert := assert.New(t)
	fn := NewDefaultBasicAuth("account", "password")
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	err := fn(c)
	assert.Equal(`basic realm="basic auth"`, c.GetHeader(elton.HeaderWWWAuthenticate))
	assert.Equal(ErrBasicAuthUnauthorized, err)
}

func TestBasicAuthValidateError(t *testing.T) {
	assert := assert.New(t)
	// 校验出错
	fn := NewBasicAuth(BasicAuthConfig{
		Validate: func(account, password string, _ *elton.Context) (bool, error) {
			return false, errors.New("custom error")
		},
	})
	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	c.Request.SetBasicAuth("account", "password")
	err := fn(c)
	assert.NotNil(err)
	he, ok := err.(*hes.Error)
	assert.True(ok)
	assert.Equal(400, he.StatusCode)
	assert.Equal("category=elton-basic-auth, message=custom error", he.Error())
}

func TestBasicAuthValidateFail(t *testing.T) {
	assert := assert.New(t)
	// 校验失败（账号或密码错误)
	fn := NewBasicAuth(BasicAuthConfig{
		Validate: func(account, password string, _ *elton.Context) (bool, error) {
			return false, nil
		},
		Realm: "custom realm",
	})

	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	c.Request.SetBasicAuth("account", "pass")
	err := fn(c)
	assert.Equal(`basic realm="custom realm"`, c.GetHeader(elton.HeaderWWWAuthenticate))
	assert.Equal(ErrBasicAuthUnauthorized, err)
}

func TestBasicAuth(t *testing.T) {
	assert := assert.New(t)
	// 校验失败（账号或密码错误)
	fn := NewDefaultBasicAuth("account", "password")

	c := elton.NewContext(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	c.Request.SetBasicAuth("account", "password")
	done := false
	c.Next = func() error {
		done = true
		return nil
	}
	err := fn(c)
	assert.Nil(err)
	assert.True(done)
}
