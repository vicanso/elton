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
	"strings"

	"github.com/vicanso/cod"
	"github.com/vicanso/hes"
)

var (
	errSubmitTooFrequently = &hes.Error{
		StatusCode: http.StatusBadRequest,
		Message:    "submit too frequently",
		Category:   "cod-concurrent-limiter",
	}
)

const (
	ipKey     = ":ip"
	headerKey = "h:"
	queryKey  = "q:"
	paramKey  = "p:"
)

type (
	// Lock lock the key
	Lock func(string, *cod.Context) (bool, func(), error)
	// ConcurrentLimiterConfig concurrent limiter config
	ConcurrentLimiterConfig struct {
		// 生成limit key的相关参数
		Keys []string
		Lock Lock
	}
	// ConcurrentKeyInfo the concurrent key's info
	ConcurrentKeyInfo struct {
		Name   string
		Params bool
		Query  bool
		Header bool
		Body   bool
		IP     bool
	}
)

// NewConcurrentLimiter create a concurrent limiter middleware
func NewConcurrentLimiter(conf ConcurrentLimiterConfig) cod.Handler {
	if conf.Lock == nil {
		panic("require lock function")
	}
	keys := make([]*ConcurrentKeyInfo, 0)
	// 根据配置生成key的处理
	for _, key := range conf.Keys {
		if key == ipKey {
			keys = append(keys, &ConcurrentKeyInfo{
				IP: true,
			})
			continue
		}
		if strings.HasPrefix(key, headerKey) {
			keys = append(keys, &ConcurrentKeyInfo{
				Name:   key[2:],
				Header: true,
			})
			continue
		}
		if strings.HasPrefix(key, queryKey) {
			keys = append(keys, &ConcurrentKeyInfo{
				Name:  key[2:],
				Query: true,
			})
			continue
		}
		if strings.HasPrefix(key, paramKey) {
			keys = append(keys, &ConcurrentKeyInfo{
				Name:   key[2:],
				Params: true,
			})
			continue
		}
		keys = append(keys, &ConcurrentKeyInfo{
			Name: key,
			Body: true,
		})
	}
	return func(c *cod.Context) (err error) {
		values := make([]string, len(keys))
		req := c.Request
		// 获取 lock 的key
		for i, key := range keys {
			v := ""
			name := key.Name
			if key.IP {
				v = c.RealIP()
			} else if key.Header {
				v = req.Header.Get(name)
			} else if key.Query {
				query := c.Query()
				v = query[name]
			} else if key.Params {
				v = c.Param(name)
			} else {
				body := c.RequestBody
				v = json.Get(body, name).ToString()
			}
			values[i] = v
		}
		lockKey := strings.Join(values, ",")

		success, unlock, err := conf.Lock(lockKey, c)
		if err != nil {
			return
		}
		if !success {
			err = errSubmitTooFrequently
			return
		}

		if unlock != nil {
			defer unlock()
		}

		return c.Next()
	}
}
