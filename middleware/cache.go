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
	"encoding/binary"
	"net/http"
)

const statusByteSize = 2
const headerBytesSize = 4

type CacheResponse struct {
	// 状态码
	StatusCode int
	// 响应头
	Header http.Header
	// 响应数据
	Body *bytes.Buffer
}

// Bytes converts the cache response to bytes
func (cp *CacheResponse) Bytes() []byte {
	headers := NewHTTPHeaders(cp.Header)
	headersLength := len(headers)
	// 2个字节保存status code
	// 4个字节保存http header长度
	// http header数据
	// body 数据

	size := statusByteSize + headerBytesSize + headersLength + cp.Body.Len()

	buf := make([]byte, size)
	offset := 0
	binary.BigEndian.PutUint16(buf, uint16(cp.StatusCode))
	offset += statusByteSize

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(headers)))
	offset += headerBytesSize

	if headersLength != 0 {
		copy(buf[offset:], headers)
		offset += headersLength
	}

	copy(buf[offset:], cp.Body.Bytes())

	return buf
}

// NewCacheResponse create a new cache response
func NewCacheResponse(data []byte) *CacheResponse {
	if len(data) < statusByteSize+headerBytesSize {
		return nil
	}
	offset := 0

	status := binary.BigEndian.Uint16(data)
	offset += statusByteSize

	size := int(binary.BigEndian.Uint32(data[offset:]))
	offset += headerBytesSize
	hs := HTTPHeaders(data[offset : offset+size])

	offset += size

	body := data[offset:]

	return &CacheResponse{
		StatusCode: int(status),
		Header:     hs.Header(),
		Body:       bytes.NewBuffer(body),
	}
}
