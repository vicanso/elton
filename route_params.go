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

package elton

// RouteParams is a structure to track URL routing parameters efficiently.
type RouteParams struct {
	Keys, Values     []string
	methodNotAllowed bool
}

// Add will append a URL parameter to the end of the route param
func (s *RouteParams) Add(key, value string) {
	s.Keys = append(s.Keys, key)
	s.Values = append(s.Values, value)
}

// Reset reset the params
func (s *RouteParams) Reset() {
	s.Keys = s.Keys[:0]
	s.Values = s.Values[:0]
	s.methodNotAllowed = false
}

// Get get param's value by key
func (s *RouteParams) Get(key string) (value string) {
	index := -1
	for i, k := range s.Keys {
		if key == k {
			index = i
			break
		}
	}
	if index != -1 {
		value = s.Values[index]
	}
	return
}

// ToMap convert route params to map[string]string
func (s *RouteParams) ToMap() map[string]string {
	m := make(map[string]string, len(s.Keys))
	for index, key := range s.Keys {
		m[key] = s.Values[index]
	}
	return m
}
