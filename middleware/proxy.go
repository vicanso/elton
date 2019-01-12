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

	"github.com/vicanso/hes"

	"github.com/vicanso/cod"
)

var (
	errTargetIsNil = hes.New("target can not be nil")
)

type (
	// TargetPicker target picker function
	TargetPicker func(c *cod.Context) (*url.URL, error)
	// ProxyConfig proxy config
	ProxyConfig struct {
		Target       *url.URL
		Host         string
		Transport    *http.Transport
		TargetPicker TargetPicker
	}
)

// NewProxy create a proxy middleware
func NewProxy(config ProxyConfig) cod.Handler {
	if config.Target == nil && config.TargetPicker == nil {
		panic("require target or targer picker")
	}
	// TODO 增强proxy的方式，可以动态选择backend
	// if config.URL == "" {
	// 	panic("require url config")
	// }
	return func(c *cod.Context) (err error) {
		target := config.Target
		if target == nil {
			target, err = config.TargetPicker(c)
			if err != nil {
				return
			}
		}
		// 如果无target，则抛错
		if target == nil {
			err = errTargetIsNil
			return
		}
		p := httputil.NewSingleHostReverseProxy(target)
		if config.Transport != nil {
			p.Transport = config.Transport
		}
		req := c.Request
		if config.Host != "" {
			req.Host = config.Host
		}
		p.ServeHTTP(c, req)
		return c.Next()
	}
}
