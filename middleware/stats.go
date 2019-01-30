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
	"net/url"
	"sync/atomic"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

type (
	// OnStats on stats function
	OnStats func(*StatsInfo, *cod.Context)
	// StatsConfig stats config
	StatsConfig struct {
		OnStats OnStats
		Skipper Skipper
	}
	// StatsInfo 统计信息
	StatsInfo struct {
		CID        string
		IP         string
		Method     string
		Route      string
		URI        string
		Status     int
		Consuming  time.Duration
		Type       int
		Size       int
		Connecting uint32
	}
)

// NewStats create a new stats middleware
func NewStats(config StatsConfig) cod.Handler {
	if config.OnStats == nil {
		panic("require on stats function")
	}
	var connectingCount uint32
	skipper := config.Skipper
	if skipper == nil {
		skipper = DefaultSkipper
	}
	return func(c *cod.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		atomic.AddUint32(&connectingCount, 1)
		defer atomic.AddUint32(&connectingCount, ^uint32(0))

		startedAt := time.Now()

		req := c.Request
		uri, _ := url.QueryUnescape(req.RequestURI)
		if uri == "" {
			uri = req.RequestURI
		}
		info := &StatsInfo{
			CID:        c.ID,
			Method:     req.Method,
			Route:      c.Route,
			URI:        uri,
			Connecting: connectingCount,
			IP:         c.RealIP(),
		}

		err = c.Next()

		info.Consuming = time.Since(startedAt)
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
		return
	}
}
