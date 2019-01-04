// Copyright 2018 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
