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
	"encoding/json"
	"errors"
	"regexp"

	"github.com/vicanso/elton"
)

const (
	// HandleSuccess handle success
	HandleSuccess = iota
	// HandleFail handle fail
	HandleFail
)

var (
	defaultTrackerMaskFields = regexp.MustCompile(`(?i)password`)
	ErrTrackerNoFunction     = errors.New("require on track function")
)

type (
	// TrackerInfo tracker info
	TrackerInfo struct {
		CID    string                 `json:"cid,omitempty"`
		Query  map[string]string      `json:"query,omitempty"`
		Params map[string]string      `json:"params,omitempty"`
		Form   map[string]interface{} `json:"form,omitempty"`
		Result int                    `json:"result,omitempty"`
		Err    error                  `json:"err,omitempty"`
	}
	// OnTrack on track function
	OnTrack func(*TrackerInfo, *elton.Context)
	// TrackerConfig tracker config
	TrackerConfig struct {
		OnTrack OnTrack
		Mask    *regexp.Regexp
		Skipper elton.Skipper
	}
)

func convertMap(data map[string]string, mask *regexp.Regexp) map[string]string {
	size := len(data)
	if size == 0 {
		return nil
	}
	m := make(map[string]string, size)
	for k, v := range data {
		if mask.MatchString(k) {
			m[k] = "***"
		} else {
			m[k] = v
		}
	}
	return m
}

// NewTracker create a tracker middleware
func NewTracker(config TrackerConfig) elton.Handler {
	mask := config.Mask
	if mask == nil {
		mask = defaultTrackerMaskFields
	}
	if config.OnTrack == nil {
		panic(ErrTrackerNoFunction)
	}
	skipper := config.Skipper
	if skipper == nil {
		skipper = elton.DefaultSkipper
	}
	return func(c *elton.Context) (err error) {
		if skipper(c) {
			return c.Next()
		}
		result := HandleSuccess
		query := convertMap(c.Query(), mask)
		params := convertMap(c.Params.ToMap(), mask)
		var form map[string]interface{}
		if len(c.RequestBody) != 0 {
			form = make(map[string]interface{})
			_ = json.Unmarshal(c.RequestBody, &form)
			for k := range form {
				if mask.MatchString(k) {
					form[k] = "***"
				}
			}
		}
		err = c.Next()
		if err != nil {
			result = HandleFail
		}
		config.OnTrack(&TrackerInfo{
			CID:    c.ID,
			Query:  query,
			Params: params,
			Form:   form,
			Result: result,
			Err:    err,
		}, c)
		return
	}
}
