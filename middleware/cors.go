// MIT License

// Copyright (c) 2026 Tree Xie

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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vicanso/elton/v2"
)

const (
	headerOrigin                        = "Origin"
	headerAccessControlRequestMethod    = "Access-Control-Request-Method"
	headerAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	headerAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	headerAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	headerAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	headerAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	headerAccessControlMaxAge           = "Access-Control-Max-Age"
	headerAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	headerVary                          = "Vary"
)

// CORSConfig CORS middleware config
type CORSConfig struct {
	// Skipper skipper function
	Skipper elton.Skipper
	// AllowOrigins allowed origins; "*" means any origin (incompatible with credentials)
	AllowOrigins []string
	// AllowOriginFunc custom origin check; if set, overrides AllowOrigins list matching
	AllowOriginFunc func(origin string) bool
	// AllowMethods allowed methods for preflight; default common REST methods
	AllowMethods []string
	// AllowHeaders allowed request headers for preflight; empty echoes request headers
	AllowHeaders []string
	// ExposeHeaders response headers browsers may expose to JS
	ExposeHeaders []string
	// MaxAge preflight cache duration
	MaxAge time.Duration
	// AllowCredentials whether to set Access-Control-Allow-Credentials
	AllowCredentials bool
}

// NewDefaultCORS returns CORS middleware allowing any origin without credentials.
func NewDefaultCORS() elton.Handler {
	return NewCORS(CORSConfig{
		AllowOrigins: []string{"*"},
	})
}

// NewCORS returns a CORS middleware.
func NewCORS(config CORSConfig) elton.Handler {
	skipper := getSkipper(config.Skipper)
	allowOrigins := config.AllowOrigins
	if len(allowOrigins) == 0 && config.AllowOriginFunc == nil {
		allowOrigins = []string{"*"}
	}
	allowMethods := config.AllowMethods
	if len(allowMethods) == 0 {
		allowMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		}
	}
	methodsHeader := strings.Join(allowMethods, ", ")
	var allowHeadersHeader string
	if len(config.AllowHeaders) != 0 {
		allowHeadersHeader = strings.Join(config.AllowHeaders, ", ")
	}
	var exposeHeadersHeader string
	if len(config.ExposeHeaders) != 0 {
		exposeHeadersHeader = strings.Join(config.ExposeHeaders, ", ")
	}
	var maxAgeHeader string
	if config.MaxAge > 0 {
		maxAgeHeader = strconv.Itoa(int(config.MaxAge.Seconds()))
	}
	allowAll := false
	originSet := make(map[string]struct{}, len(allowOrigins))
	for _, o := range allowOrigins {
		if o == "*" {
			allowAll = true
			continue
		}
		originSet[o] = struct{}{}
	}

	matchOrigin := func(origin string) (string, bool) {
		if origin == "" {
			return "", false
		}
		if config.AllowOriginFunc != nil {
			if config.AllowOriginFunc(origin) {
				return origin, true
			}
			return "", false
		}
		if allowAll {
			// credentials 时不能回写 *，需回显具体 origin
			if config.AllowCredentials {
				return origin, true
			}
			return "*", true
		}
		if _, ok := originSet[origin]; ok {
			return origin, true
		}
		return "", false
	}

	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		origin := c.GetRequestHeader(headerOrigin)
		allowOrigin, ok := matchOrigin(origin)
		if !ok {
			// 非 CORS 请求或 origin 不允许：继续业务（不设置 ACAO）
			return c.Next()
		}

		// 实际请求 / 预检均设置允许的 origin
		c.SetHeader(headerAccessControlAllowOrigin, allowOrigin)
		if allowOrigin != "*" {
			c.AddHeader(headerVary, headerOrigin)
		}
		if config.AllowCredentials {
			c.SetHeader(headerAccessControlAllowCredentials, "true")
		}
		if exposeHeadersHeader != "" {
			c.SetHeader(headerAccessControlExposeHeaders, exposeHeadersHeader)
		}

		// preflight
		if c.Request.Method == http.MethodOptions &&
			c.GetRequestHeader(headerAccessControlRequestMethod) != "" {
			c.SetHeader(headerAccessControlAllowMethods, methodsHeader)
			reqHeaders := c.GetRequestHeader(headerAccessControlRequestHeaders)
			if allowHeadersHeader != "" {
				c.SetHeader(headerAccessControlAllowHeaders, allowHeadersHeader)
			} else if reqHeaders != "" {
				c.SetHeader(headerAccessControlAllowHeaders, reqHeaders)
			}
			if maxAgeHeader != "" {
				c.SetHeader(headerAccessControlMaxAge, maxAgeHeader)
			}
			c.NoContent()
			return nil
		}

		return c.Next()
	}
}
