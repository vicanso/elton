// MIT License

// Copyright (c) 2026 Tree Xie

package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

func TestTimeout(t *testing.T) {
	t.Run("panic on invalid timeout", func(t *testing.T) {
		assert := assert.New(t)
		assert.Panics(func() {
			NewTimeout(TimeoutConfig{Timeout: 0})
		})
	})

	t.Run("no timeout", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultTimeout(time.Second)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error {
			// context 应带 deadline
			_, ok := c.Context().Deadline()
			assert.True(ok)
			return nil
		}
		err := fn(c)
		assert.Nil(err)
	})

	t.Run("deadline exceeded returns 504", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewTimeout(TimeoutConfig{Timeout: 20 * time.Millisecond})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error {
			time.Sleep(50 * time.Millisecond)
			return c.Context().Err()
		}
		err := fn(c)
		assert.Equal(ErrRequestTimeout, err)
		he := &hes.Error{}
		assert.True(errors.As(err, &he))
		assert.Equal(http.StatusGatewayTimeout, he.StatusCode)
	})

	t.Run("business error after deadline still returned", func(t *testing.T) {
		assert := assert.New(t)
		bizErr := errors.New("biz fail")
		fn := NewTimeout(TimeoutConfig{Timeout: 10 * time.Millisecond})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error {
			time.Sleep(30 * time.Millisecond)
			return bizErr
		}
		err := fn(c)
		assert.Equal(bizErr, err)
	})

	t.Run("custom timeout error", func(t *testing.T) {
		assert := assert.New(t)
		custom := errors.New("slow")
		fn := NewTimeout(TimeoutConfig{
			Timeout: 10 * time.Millisecond,
			Error:   custom,
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error {
			<-c.Context().Done()
			return context.DeadlineExceeded
		}
		err := fn(c)
		assert.Equal(custom, err)
	})

	t.Run("skipper", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewTimeout(TimeoutConfig{
			Timeout: time.Millisecond,
			Skipper: func(*elton.Context) bool { return true },
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error {
			_, ok := c.Context().Deadline()
			assert.False(ok)
			return nil
		}
		assert.Nil(fn(c))
	})
}
