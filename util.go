// MIT License

// Copyright (c) 2022 Tree Xie

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

import (
	"bytes"
	"io"
	"sync"
)

// A BufferPool is an interface for getting and
// returning temporary buffer
type BufferPool interface {
	Get() *bytes.Buffer
	Put(*bytes.Buffer)
}

type simpleBufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a buffer pool, if the init cap gt 0,
// the buffer will be init with cap size
func NewBufferPool(initCap int) BufferPool {
	p := &simpleBufferPool{}
	p.pool.New = func() interface{} {
		if initCap > 0 {
			return bytes.NewBuffer(make([]byte, 0, initCap))
		}
		return &bytes.Buffer{}
	}
	return p
}

func (sp *simpleBufferPool) Get() *bytes.Buffer {
	buf := sp.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (sp *simpleBufferPool) Put(buf *bytes.Buffer) {
	sp.pool.Put(buf)
}

// copy from io.ReadAll
// ReadAll reads from r until an error or EOF and returns the data it read.
// A successful call returns err == nil, not err == EOF. Because ReadAll is
// defined to read from src until EOF, it does not treat an EOF from Read
// as an error to be reported.
func ReadAllInitCap(r io.Reader, initCap int) ([]byte, error) {
	if initCap <= 0 {
		initCap = 512
	}
	b := make([]byte, 0, initCap)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
	}
}

// ReadAllToBuffer reader from r util an error or EOF and write data to buffer.
// A successful call returns err == nil, not err == EOF. Because ReadAll is
// defined to read from src until EOF, it does not treat an EOF from Read
// as an error to be reported.
func ReadAllToBuffer(r io.Reader, buffer *bytes.Buffer) error {
	b := make([]byte, 512)
	for {
		n, err := r.Read(b)
		// 先将读取数据写入
		buffer.Write(b[0:n])
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
	}
}
