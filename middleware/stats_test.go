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
	"net/http"
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

func TestSkip(t *testing.T) {
	assert := assert.New(t)
	fn := NewStats(StatsConfig{
		OnStats: func(info *StatsInfo, _ *elton.Context) {

		},
	})
	c := elton.NewContext(nil, nil)
	done := false
	c.Next = func() error {
		done = true
		return nil
	}
	c.Committed = true
	err := fn(c)
	assert.Nil(err)
	assert.True(done)
}

func TestStats(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "http://127.0.0.1/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		c.BodyBuffer = bytes.NewBufferString("abcd")
		done := false
		fn := NewStats(StatsConfig{
			OnStats: func(info *StatsInfo, _ *elton.Context) {
				assert.Equal(http.StatusOK, info.Status, "status code should be 200")
				done = true
			},
		})
		c.Next = func() error {
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.True(done)
	})

	t.Run("return hes error", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "http://127.0.0.1/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		done := false
		fn := NewStats(StatsConfig{
			OnStats: func(info *StatsInfo, _ *elton.Context) {
				assert.Equal(http.StatusBadRequest, info.Status)
				done = true
			},
		})
		c.Next = func() error {
			return hes.New("abc")
		}
		err := fn(c)
		assert.NotNil(err)
		assert.True(done, "on stats shouldn be called when return error")
	})

	t.Run("return normal error", func(t *testing.T) {
		assert := assert.New(t)
		req := httptest.NewRequest("GET", "http://127.0.0.1/users/me", nil)
		resp := httptest.NewRecorder()
		c := elton.NewContext(resp, req)
		done := false
		fn := NewStats(StatsConfig{
			OnStats: func(info *StatsInfo, _ *elton.Context) {
				assert.Equal(http.StatusInternalServerError, info.Status)
				done = true
			},
		})
		c.Next = func() error {
			return errors.New("abc")
		}
		err := fn(c)
		assert.NotNil(err)
		assert.True(done, "on stats shouldn be called when return error")
	})
}
