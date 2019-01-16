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
	"regexp"
	"strconv"
	"strings"

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
		Rewrites     []string
		Host         string
		Transport    *http.Transport
		TargetPicker TargetPicker
		Skipper      Skipper
	}
)

func captureTokens(pattern *regexp.Regexp, input string) *strings.Replacer {
	groups := pattern.FindAllStringSubmatch(input, -1)
	if groups == nil {
		return nil
	}
	values := groups[0][1:]
	replace := make([]string, 2*len(values))
	for i, v := range values {
		j := 2 * i
		replace[j] = "$" + strconv.Itoa(i+1)
		replace[j+1] = v
	}
	return strings.NewReplacer(replace...)
}

func rewrite(rewriteRegexp map[*regexp.Regexp]string, req *http.Request) {
	for k, v := range rewriteRegexp {
		replacer := captureTokens(k, req.URL.Path)
		if replacer != nil {
			req.URL.Path = replacer.Replace(v)
		}
	}
}

// NewProxy create a proxy middleware
func NewProxy(config ProxyConfig) cod.Handler {
	if config.Target == nil && config.TargetPicker == nil {
		panic("require target or targer picker")
	}
	regs, err := cod.GenerateRewrites(config.Rewrites)
	if err != nil {
		panic(err)
	}
	skiper := config.Skipper
	if skiper == nil {
		skiper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skiper(c) {
			return c.Next()
		}
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
		var originalPath, originalHost string
		if regs != nil {
			originalPath = req.URL.Path
			rewrite(regs, req)
		}
		if config.Host != "" {
			originalHost = req.Host
			req.Host = config.Host
		}
		p.ServeHTTP(c, req)
		if originalPath != "" {
			req.URL.Path = originalPath
		}
		if originalHost != "" {
			req.Host = originalHost
		}
		return c.Next()
	}
}
