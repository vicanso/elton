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
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

const (
	// ErrProxyCategory proxy error category
	ErrProxyCategory = "elton-proxy"
)

var (
	// ErrProxyTargetIsNil target is nil
	ErrProxyTargetIsNil = &hes.Error{
		Exception:  true,
		Message:    "target can not be nil",
		StatusCode: http.StatusBadRequest,
		Category:   ErrProxyCategory,
	}
	ErrProxyNoTargetFunction = errors.New("require target or targer picker")
)

type (
	// ProxyDone http proxy done callback
	ProxyDone func(*elton.Context)
	// ProxyTargetPicker target picker function
	ProxyTargetPicker func(c *elton.Context) (*url.URL, ProxyDone, error)
	// Config proxy config
	ProxyConfig struct {
		// Done proxy done callback
		Done         ProxyDone
		Target       *url.URL
		Rewrites     []string
		Host         string
		Transport    *http.Transport
		TargetPicker ProxyTargetPicker
		Skipper      elton.Skipper
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

func urlRewrite(rewriteRegexp map[*regexp.Regexp]string, req *http.Request) {
	for k, v := range rewriteRegexp {
		replacer := captureTokens(k, req.URL.Path)
		if replacer != nil {
			req.URL.Path = replacer.Replace(v)
		}
	}
}

// generateRewrites generate rewrites
func generateRewrites(rewrites []string) (m map[*regexp.Regexp]string, err error) {
	if len(rewrites) == 0 {
		return
	}
	m = make(map[*regexp.Regexp]string)

	for _, value := range rewrites {
		arr := strings.Split(value, ":")
		if len(arr) != 2 {
			continue
		}
		k := arr[0]
		v := arr[1]
		k = strings.Replace(k, "*", "(\\S*)", -1)
		reg, e := regexp.Compile(k)
		if e != nil {
			err = e
			break
		}
		m[reg] = v
	}
	return
}

// NewProxy create a proxy middleware
func NewProxy(config ProxyConfig) elton.Handler {
	if config.Target == nil && config.TargetPicker == nil {
		panic(ErrProxyNoTargetFunction)
	}
	regs, err := generateRewrites(config.Rewrites)
	if err != nil {
		panic(err)
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		target := config.Target
		var done ProxyDone
		if target == nil {
			target, done, err = config.TargetPicker(c)
			if err != nil {
				return
			}
		}
		// 如果无target，则抛错
		if target == nil {
			err = ErrProxyTargetIsNil
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
			urlRewrite(regs, req)
		}
		if config.Host != "" {
			originalHost = req.Host
			req.Host = config.Host
		}
		p.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, e error) {
			he := hes.NewWithError(e)
			he.Category = ErrProxyCategory
			he.Exception = true
			err = he
		}
		p.ServeHTTP(c, req)
		if config.Done != nil {
			config.Done(c)
		}
		if done != nil {
			done(c)
		}
		if err != nil {
			return
		}
		if originalPath != "" {
			req.URL.Path = originalPath
		}
		if originalHost != "" {
			req.Host = originalHost
		}
		return c.Next()
	}
}