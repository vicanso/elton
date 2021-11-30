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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vicanso/elton"
)

func TestNoTrackPanic(t *testing.T) {
	assert := assert.New(t)
	done := false
	defer func() {
		r := recover()
		assert.Equal(ErrTrackerNoFunction, r.(error))
		done = true
	}()

	NewTracker(TrackerConfig{})
	assert.True(done)
}

func TestConvertMap(t *testing.T) {
	assert := assert.New(t)
	assert.Nil(convertMap(nil, nil, 0))
	assert.Equal(map[string]string{
		"foo":      "b ... (2 more)",
		"password": "***",
	}, convertMap(map[string]string{
		"password": "123",
		"foo":      "bar",
	}, defaultTrackerMaskFields, 1))
}

func TestTracker(t *testing.T) {
	assert := assert.New(t)

	skipErr := errors.New("skip error")
	// next直接返回skip error，用于判断是否执行了next
	next := func() error {
		return skipErr
	}

	trackerInfoKey := "_trackerInfo"
	defaultTracker := NewTracker(TrackerConfig{
		OnTrack: func(info *TrackerInfo, c *elton.Context) {
			c.Set(trackerInfoKey, info)
		},
	})
	tests := []struct {
		newContext func() *elton.Context
		err        error
		info       *TrackerInfo
	}{
		{
			newContext: func() *elton.Context {
				req := httptest.NewRequest("POST", "/users/login?type=1&passwordType=2", nil)
				c := elton.NewContext(nil, req)
				c.RequestBody = []byte(`{
		"account": "tree.xie tree.xie tree.xie",
		"password": "password"
	}`)
				c.Params = new(elton.RouteParams)
				c.Params.Add("category", "login")
				c.Next = next
				return c
			},
			err: skipErr,
			info: &TrackerInfo{
				Result: HandleFail,
				Query: map[string]string{
					"type":         "1",
					"passwordType": "***",
				},
				Params: map[string]string{
					"category": "login",
				},
				Form: map[string]interface{}{
					"account":  "tree.xie tree.xie tr ... (6 more)",
					"password": "***",
				},
				Err: skipErr,
			},
		},
	}

	for _, tt := range tests {
		c := tt.newContext()
		err := defaultTracker(c)
		assert.Equal(tt.err, err)
		v, ok := c.Get(trackerInfoKey)
		assert.True(ok)
		info, ok := v.(*TrackerInfo)
		assert.True(ok)
		assert.NotEmpty(info.Latency)
		// 重置耗时
		info.Latency = 0
		assert.Equal(tt.info, info)
	}
}
