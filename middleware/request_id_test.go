// MIT License

// Copyright (c) 2026 Tree Xie

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton/v2"
)

func TestRequestID(t *testing.T) {
	t.Run("generate new id", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultRequestID()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error { return nil }
		assert.Nil(fn(c))
		id := GetRequestID(c)
		assert.Len(id, 32)
		assert.Equal(id, c.GetHeader(HeaderXRequestID))
		assert.Equal(id, c.GetRequestHeader(HeaderXRequestID))
		assert.Equal(id, c.ID)
	})

	t.Run("reuse inbound header", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultRequestID()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderXRequestID, "client-id-1")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error { return nil }
		assert.Nil(fn(c))
		assert.Equal("client-id-1", GetRequestID(c))
		assert.Equal("client-id-1", c.GetHeader(HeaderXRequestID))
	})

	t.Run("custom generator and header", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewRequestID(RequestIDConfig{
			Header:     "X-Trace-Id",
			ContextKey: "traceId",
			Generator:  func() string { return "fixed-id" },
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error { return nil }
		assert.Nil(fn(c))
		assert.Equal("fixed-id", elton.GetContextValue[string](c, "traceId"))
		assert.Equal("fixed-id", c.GetHeader("X-Trace-Id"))
	})

	t.Run("disable response header", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewRequestID(RequestIDConfig{
			Generator:             func() string { return "only-ctx" },
			DisableResponseHeader: true,
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error { return nil }
		assert.Nil(fn(c))
		assert.Equal("only-ctx", GetRequestID(c))
		assert.Empty(c.GetHeader(HeaderXRequestID))
	})

	t.Run("does not override existing c.ID", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewRequestID(RequestIDConfig{
			Generator: func() string { return "generated" },
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.ID = "existing"
		c.Next = func() error { return nil }
		assert.Nil(fn(c))
		assert.Equal("existing", c.ID)
		assert.Equal("generated", GetRequestID(c))
	})

	t.Run("skipper", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewRequestID(RequestIDConfig{
			Skipper: func(*elton.Context) bool { return true },
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = func() error { return nil }
		assert.Nil(fn(c))
		assert.Empty(GetRequestID(c))
	})
}
