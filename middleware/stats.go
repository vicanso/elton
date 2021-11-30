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
	"net/url"
	"sync/atomic"
	"time"

	"github.com/vicanso/elton"
	"github.com/vicanso/hes"
)

var (
	ErrStatsNoFunction = errors.New("require on stats function")
)

type (
	// OnStats on stats function
	OnStats func(*StatsInfo, *elton.Context)
	// StatsConfig stats config
	StatsConfig struct {
		OnStats OnStats
		Skipper elton.Skipper
	}
	// StatsInfo stats's info
	StatsInfo struct {
		CID             string        `json:"cid,omitempty"`
		IP              string        `json:"ip,omitempty"`
		Method          string        `json:"method,omitempty"`
		Route           string        `json:"route,omitempty"`
		URI             string        `json:"uri,omitempty"`
		Status          int           `json:"status,omitempty"`
		Latency         time.Duration `json:"latency,omitempty"`
		Type            int           `json:"type,omitempty"`
		RequestBodySize int           `json:"requestBodySize"`
		Size            int           `json:"size,omitempty"`
		Connecting      uint32        `json:"connecting,omitempty"`
	}
)

// NewStats returns a new stats middleware,
// it will throw a panic if the OnStats is nil.
func NewStats(config StatsConfig) elton.Handler {
	if config.OnStats == nil {
		panic(ErrStatsNoFunction)
	}
	var connectingCount uint32
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) error {
		if skipper(c) {
			return c.Next()
		}
		connecting := atomic.AddUint32(&connectingCount, 1)
		defer atomic.AddUint32(&connectingCount, ^uint32(0))

		startedAt := time.Now()

		req := c.Request
		uri, _ := url.QueryUnescape(req.RequestURI)
		if uri == "" {
			uri = req.RequestURI
		}
		info := &StatsInfo{
			CID:             c.ID,
			Method:          req.Method,
			Route:           c.Route,
			URI:             uri,
			Connecting:      connecting,
			IP:              c.RealIP(),
			RequestBodySize: len(c.RequestBody),
		}

		err := c.Next()

		info.Latency = time.Since(startedAt)
		status := c.StatusCode
		if err != nil {
			he, ok := err.(*hes.Error)
			if ok {
				status = he.StatusCode
			} else {
				status = http.StatusInternalServerError
			}
		}
		if status == 0 {
			status = http.StatusOK
		}
		info.Status = status
		info.Type = status / 100
		size := 0
		if c.BodyBuffer != nil {
			size = c.BodyBuffer.Len()
		}
		info.Size = size

		config.OnStats(info, c)
		return err
	}
}
