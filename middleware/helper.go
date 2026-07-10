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
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/hes"
)

// getSkipper returns the skipper if it's not nil,
// otherwise returns elton.DefaultSkipper
func getSkipper(skipper elton.Skipper) elton.Skipper {
	if skipper == nil {
		return elton.DefaultSkipper
	}
	return skipper
}

// wrapAsHesError returns the *hes.Error of err if err's chain contains one,
// otherwise wraps err as an internal server error exception with the category
func wrapAsHesError(err error, category string) *hes.Error {
	he := &hes.Error{}
	if errors.As(err, &he) {
		return he
	}
	he = hes.Wrap(err)
	he.StatusCode = http.StatusInternalServerError
	he.Exception = true
	he.Category = category
	return he
}

// genETag generates an etag of the buffer by sha1
func genETag(buf []byte) string {
	size := len(buf)
	if size == 0 {
		return `"0-2jmj7l5rSw0yVb_vlWAYkK_YBwk="`
	}
	h := sha1.New()
	h.Write(buf)
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf(`"%x-%s"`, size, hash)
}
