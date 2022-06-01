// MIT License

// Copyright (c) 2022 Tree Xie

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
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/vicanso/elton"
)

// NewPrefixURL returns a new prefix url handler, it will replace the prefix of url as ""
func NewPrefixURL(prefix ...string) elton.PreHandler {
	var reg *regexp.Regexp
	if len(prefix) != 0 {
		reg = regexp.MustCompile(fmt.Sprintf(`^(%s)`, strings.Join(prefix, "|")))
	}
	return func(req *http.Request) {
		if reg == nil {
			return
		}
		req.URL.Path = reg.ReplaceAllString(req.URL.Path, "")
	}
}
