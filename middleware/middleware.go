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
	"bytes"
	"compress/gzip"

	jsoniter "github.com/json-iterator/go"
	"github.com/vicanso/cod"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

const (
	// ErrCategoryBasicAuth basic auth error category
	ErrCategoryBasicAuth = "cod-basic-auth"
	// ErrCategoryBodyParser body parser error category
	ErrCategoryBodyParser = "cod-body-parser"
	// ErrCategoryConcurrentLimiter concurrent limiter error category
	ErrCategoryConcurrentLimiter = "cod-concurrent-limiter"
	// ErrCategoryProxy proxy error category
	ErrCategoryProxy = "cod-proxy"
	// ErrCategoryResponder responder error category
	ErrCategoryResponder = "cod-responder"
	// ErrCategorySession session error category
	ErrCategorySession = "cod-session"
	// ErrCategoryStaticServe static serve error category
	ErrCategoryStaticServe = "cod-static-serve"
	// ErrCategoryRecover recover error category
	ErrCategoryRecover = "cod-recover"
)

type (
	// Skipper check for skip middleware
	Skipper func(c *cod.Context) bool
)

// DefaultSkipper default skipper function(not skip)
func DefaultSkipper(c *cod.Context) bool {
	return c.Committed
}

// doGzip 对数据压缩
func doGzip(buf []byte, level int) ([]byte, error) {
	var b bytes.Buffer
	if level <= 0 {
		level = gzip.DefaultCompression
	}
	w, _ := gzip.NewWriterLevel(&b, level)
	_, err := w.Write(buf)
	if err != nil {
		return nil, err
	}
	w.Close()
	return b.Bytes(), nil
}
