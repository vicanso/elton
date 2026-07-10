// MIT License

// Copyright (c) 2021 Tree Xie

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
	"bytes"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
)

// 本文件实现http header的紧凑二进制编码，用于减少缓存中间件的存储空间。
//
// 单个header的编码格式（HTTPHeader）:
//   - 短头（名称在short header字典中）: [索引字节(1..127)][值]
//   - 非短头: [255][名称][":"][值]
//
// 多个值以 "\n" 连接为单个值段；多个header之间以字节0（headersJoinSep）连接为HTTPHeaders。
// 因此索引0保留作分隔符，短头索引从1开始，最多127个。
// 编码假定header的名称与值不含字节0与"\n"（Go的http库会拒绝此类非法值）。

// MaxShortHeader 短头索引的上限（不含），编码首字节小于该值表示短头索引
const MaxShortHeader = uint8(128)

// NoneMatchHeader 表示名称不在短头字典中的标记字节
const NoneMatchHeader = math.MaxUint8

const valueSep = "\n"
const headerValueSep = ":"

// headersJoinSep headers拼接的分隔符
var headersJoinSep = []byte{0}

// shortHeaderIndexes 短头字典，使用atomic保证运行期调用SetShortHeaders的并发安全。
// 注意：短头索引会被持久化至缓存数据中，运行期修改字典会导致
// 旧缓存条目解码出错误的header名称，建议仅在启动期调用SetShortHeaders。
var shortHeaderIndexes atomic.Pointer[headerIndexes]
var DefaultShortHeaders = []string{
	"accept-charset",
	"accept-encoding",
	"accept-language",
	"accept-ranges",
	"accept",
	"access-control-allow-origin",
	"age",
	"allow",
	"authorization",
	"cache-control",
	"content-disposition",
	"content-encoding",
	"content-language",
	"content-length",
	"content-location",
	"content-range",
	"content-type",
	"cookie",
	"date",
	"etag",
	"expect",
	"expires",
	"from",
	"host",
	"if-match",
	"if-modified-since",
	"if-none-match",
	"if-range",
	"if-unmodified-since",
	"last-modified",
	"link",
	"location",
	"max-forwards",
	"proxy-authenticate",
	"proxy-authorization",
	"range",
	"referer",
	"refresh",
	"retry-after",
	"server",
	"set-cookie",
	"strict-transport-security",
	"transfer-encoding",
	"user-agent",
	"vary",
	"via",
	"www-authenticate",
}

func init() {
	SetShortHeaders(DefaultShortHeaders)
}

type headerIndexes struct {
	// http头的值
	values []string
	// 名称与索引对照
	indexes map[string]uint8
	// 最大值
	max int
}

// 获取http头的名称
func (h *headerIndexes) getName(index int) string {
	if index >= h.max {
		return ""
	}
	return h.values[index]
}

// 根据值获取其对应的index
func (h *headerIndexes) getIndex(name string) (uint8, bool) {
	index, ok := h.indexes[strings.ToLower(name)]
	return index, ok
}

// SetShortHeaders sets the short header of http header
func SetShortHeaders(names []string) {
	if len(names) >= int(MaxShortHeader) {
		panic(fmt.Sprintf("the count of short header should be less than %d", int(MaxShortHeader)))
	}
	arr := make([]string, len(names)+1)
	indexes := make(map[string]uint8)
	for i, name := range names {
		// 不使用0，0用于分隔使用
		index := i + 1
		value := strings.ToLower(strings.TrimSpace(name))
		arr[index] = value
		indexes[value] = uint8(index)
	}
	shortHeaderIndexes.Store(&headerIndexes{
		values:  arr,
		indexes: indexes,
		max:     len(arr),
	})
}

// http头，[type, data]
type HTTPHeader []byte

// http头列表，使用字节0分隔[HTTPHeader 0 HTTPHeader]
type HTTPHeaders []byte

// 转换header的值，多个值以\n分隔
func toValues(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	arr := bytes.Split(data, []byte(valueSep))
	result := make([]string, len(arr))
	for index, item := range arr {
		result[index] = string(item)
	}
	return result
}

// Header converts bytes to http header(string:[]string)
func (h HTTPHeader) Header() (string, []string) {
	if len(h) == 0 {
		return "", nil
	}
	headerType := h[0]
	data := h[1:]
	if headerType < MaxShortHeader {
		name := shortHeaderIndexes.Load().getName(int(headerType))
		return name, toValues(data)
	}
	index := bytes.IndexByte(data, byte(headerValueSep[0]))
	if index < 0 {
		return "", nil
	}
	name := string(data[0:index])
	// 因为有分隔符，因此+1
	return name, toValues(data[index+1:])
}

// NewHTTPHeader new a http header
func NewHTTPHeader(name string, values []string) HTTPHeader {
	buffer := bytes.Buffer{}
	buffer.Grow(64)
	index, ok := shortHeaderIndexes.Load().getIndex(name)
	value := strings.Join(values, valueSep)
	if ok {
		buffer.WriteByte(index)
		buffer.WriteString(value)
		return buffer.Bytes()
	}
	buffer.WriteByte(NoneMatchHeader)
	buffer.WriteString(name)
	buffer.WriteString(headerValueSep)
	buffer.WriteString(value)
	return buffer.Bytes()
}

// NewHTTPHeaders new a http headers
func NewHTTPHeaders(header http.Header, ignoreHeaders ...string) HTTPHeaders {
	size := len(header)
	if size == 0 {
		return nil
	}
	arr := make([][]byte, 0, size)
	// 按小写精确匹配，避免子串误伤（如自定义的Encoding头）
	ignore := make(map[string]bool, len(ignoreHeaders))
	for _, name := range ignoreHeaders {
		ignore[strings.ToLower(name)] = true
	}
	for name, values := range header {
		if ignore[strings.ToLower(name)] {
			continue
		}
		arr = append(arr, NewHTTPHeader(name, values))
	}
	return bytes.Join(arr, headersJoinSep)
}

// Header convert to http.Header
func (hs HTTPHeaders) Header() http.Header {
	h := make(http.Header)
	for _, item := range bytes.Split(hs, headersJoinSep) {
		name, values := HTTPHeader(item).Header()
		if name == "" {
			continue
		}
		for _, value := range values {
			h.Add(name, value)
		}
	}
	return h
}
