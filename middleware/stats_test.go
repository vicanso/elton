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
	"bytes"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

func TestNoStatsPanic(t *testing.T) {
	assert := assert.New(t)
	done := false
	defer func() {
		r := recover()
		assert.Equal(r.(error), ErrStatsNoFunction)
		done = true
	}()
	NewStats(StatsConfig{})
	assert.True(done)
}

func TestStats(t *testing.T) {
	assert := assert.New(t)

	statsKey := "_stats"
	defaultStats := NewStats(StatsConfig{
		OnStats: func(info *StatsInfo, c *elton.Context) {
			c.Set(statsKey, info)
		},
	})

	tests := []struct {
		newContext func() *elton.Context
		err        error
		statusCode int
	}{
		// http error
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = func() error {
					return hes.New("abc")
				}
				return c
			},
			err:        hes.New("abc"),
			statusCode: 400,
		},
		// error
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.Next = func() error {
					return errors.New("abc")
				}
				return c
			},
			err:        errors.New("abc"),
			statusCode: 500,
		},
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("GET", "/", nil)
				resp := httptest.NewRecorder()
				c := elton.NewContext(resp, req)
				c.BodyBuffer = bytes.NewBufferString("abcd")
				c.Next = func() error {
					return nil
				}
				return c
			},
			statusCode: 200,
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := defaultStats(c)
		assert.Equal(tt.err, err)
		v, ok := c.Get(statsKey)
		assert.True(ok)
		info, ok := v.(*StatsInfo)
		assert.True(ok)
		assert.Equal(tt.statusCode, info.Status)

	}
}
