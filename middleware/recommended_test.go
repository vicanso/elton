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

func TestRecommended(t *testing.T) {
	assert := assert.New(t)
	stack := Recommended()
	assert.GreaterOrEqual(len(stack), 5)

	e := elton.NewWithoutServer()
	e.Use(stack...)
	e.GET("/hello", func(c *elton.Context) error {
		c.Body = map[string]string{"msg": "ok"}
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("Accept", "application/json")
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)

	assert.Equal(http.StatusOK, resp.Code)
	assert.Contains(resp.Header().Get(elton.HeaderContentType), "application/json")
	assert.NotEmpty(resp.Header().Get(HeaderXRequestID))
	assert.Contains(resp.Body.String(), "ok")
}

func TestRecommendedErrorPath(t *testing.T) {
	assert := assert.New(t)
	e := elton.NewWithoutServer()
	e.Use(Recommended()...)
	e.GET("/err", func(c *elton.Context) error {
		return ErrRequestTimeout
	})

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	req.Header.Set("Accept", "application/json")
	resp := httptest.NewRecorder()
	e.ServeHTTP(resp, req)
	assert.Equal(http.StatusGatewayTimeout, resp.Code)
}
