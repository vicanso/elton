package middleware

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/vicanso/cod"
)

type (
	// ProxyConfig proxy config
	ProxyConfig struct {
		URL       string
		Host      string
		Transport *http.Transport
		Next      bool
	}
)

// NewProxy create a proxy middleware
func NewProxy(config ProxyConfig) cod.Handler {
	if config.URL == "" {
		panic("require url config")
	}
	target, err := url.Parse(config.URL)
	if err != nil {
		panic(err)
	}
	return func(c *cod.Context) (err error) {
		p := httputil.NewSingleHostReverseProxy(target)
		if config.Transport != nil {
			p.Transport = config.Transport
		}
		req := c.Request
		if config.Host != "" {
			req.Host = config.Host
		}
		p.ServeHTTP(c.Response, req)
		c.Committed = true
		if config.Next {
			return c.Next()
		}
		return
	}
}
