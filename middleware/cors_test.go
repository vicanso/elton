// MIT License

// Copyright (c) 2026 Tree Xie

package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton/v2"
)

func TestCORS(t *testing.T) {
	skipErr := errors.New("skip")
	next := func() error { return skipErr }

	t.Run("default allow any origin", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewDefaultCORS()
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Origin", "https://app.example")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = next
		err := fn(c)
		assert.Equal(skipErr, err)
		assert.Equal("*", c.GetHeader(headerAccessControlAllowOrigin))
	})

	t.Run("specific origin with credentials", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewCORS(CORSConfig{
			AllowOrigins:     []string{"https://app.example"},
			AllowCredentials: true,
			ExposeHeaders:    []string{"X-Request-Id"},
		})
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Origin", "https://app.example")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = next
		_ = fn(c)
		assert.Equal("https://app.example", c.GetHeader(headerAccessControlAllowOrigin))
		assert.Equal("true", c.GetHeader(headerAccessControlAllowCredentials))
		assert.Equal("X-Request-Id", c.GetHeader(headerAccessControlExposeHeaders))
		assert.Contains(c.Header().Values(headerVary), headerOrigin)
	})

	t.Run("disallowed origin", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewCORS(CORSConfig{
			AllowOrigins: []string{"https://app.example"},
		})
		req := httptest.NewRequest(http.MethodGet, "/api", nil)
		req.Header.Set("Origin", "https://evil.example")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = next
		err := fn(c)
		assert.Equal(skipErr, err)
		assert.Empty(c.GetHeader(headerAccessControlAllowOrigin))
	})

	t.Run("preflight", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewCORS(CORSConfig{
			AllowOrigins: []string{"https://app.example"},
			AllowMethods: []string{http.MethodPost, http.MethodGet},
			AllowHeaders: []string{"Content-Type", "X-Token"},
			MaxAge:       time.Hour,
		})
		req := httptest.NewRequest(http.MethodOptions, "/api", nil)
		req.Header.Set("Origin", "https://app.example")
		req.Header.Set(headerAccessControlRequestMethod, http.MethodPost)
		req.Header.Set(headerAccessControlRequestHeaders, "Content-Type")
		c := elton.NewContext(httptest.NewRecorder(), req)
		called := false
		c.Next = func() error {
			called = true
			return nil
		}
		err := fn(c)
		assert.Nil(err)
		assert.False(called, "preflight should not call Next")
		assert.Equal(http.StatusNoContent, c.StatusCode)
		assert.Equal("POST, GET", c.GetHeader(headerAccessControlAllowMethods))
		assert.Equal("Content-Type, X-Token", c.GetHeader(headerAccessControlAllowHeaders))
		assert.Equal("3600", c.GetHeader(headerAccessControlMaxAge))
	})

	t.Run("allow origin func", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewCORS(CORSConfig{
			AllowOriginFunc: func(o string) bool {
				return o == "https://ok.example"
			},
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://ok.example")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = next
		_ = fn(c)
		assert.Equal("https://ok.example", c.GetHeader(headerAccessControlAllowOrigin))
	})

	t.Run("skipper", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewCORS(CORSConfig{
			AllowOrigins: []string{"*"},
			Skipper:      func(c *elton.Context) bool { return true },
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://x.com")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = next
		_ = fn(c)
		assert.Empty(c.GetHeader(headerAccessControlAllowOrigin))
	})

	t.Run("credentials with star echoes origin", func(t *testing.T) {
		assert := assert.New(t)
		fn := NewCORS(CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowCredentials: true,
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://app.example")
		c := elton.NewContext(httptest.NewRecorder(), req)
		c.Next = next
		_ = fn(c)
		assert.Equal("https://app.example", c.GetHeader(headerAccessControlAllowOrigin))
	})
}
