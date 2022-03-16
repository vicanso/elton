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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferPool(t *testing.T) {
	assert := assert.New(t)

	bp := NewBufferPool(512)

	buf := bp.Get()
	assert.NotNil(buf)
	assert.Equal(0, buf.Len())
	buf.WriteString("abc")
	bp.Put(buf)

	buf = bp.Get()
	assert.NotNil(buf)
	assert.Equal(0, buf.Len())
}

func TestReadAllInitCap(t *testing.T) {
	assert := assert.New(t)

	buf := &bytes.Buffer{}
	for i := 0; i < 1024*1024; i++ {
		buf.Write([]byte("hello world!"))
	}
	result := buf.Bytes()

	data, err := ReadAllInitCap(buf, 1024*100)
	assert.Nil(err)
	assert.Equal(result, data)

	data, err = ReadAllInitCap(bytes.NewBufferString("hello world!"), 1024*100)
	assert.Nil(err)
	assert.Equal([]byte("hello world!"), data)
}

func TestReadAllToBuffer(t *testing.T) {
	assert := assert.New(t)

	source := &bytes.Buffer{}
	for i := 0; i < 1024*1024; i++ {
		source.Write([]byte("hello world!"))
	}
	sourceBytes := source.Bytes()

	buf := bytes.Buffer{}
	err := ReadAllToBuffer(source, &buf)
	assert.Nil(err)
	assert.Equal(sourceBytes, buf.Bytes())

	buf.Reset()
	err = ReadAllToBuffer(bytes.NewBufferString("hello world!"), &buf)
	assert.Nil(err)
	assert.Equal([]byte("hello world!"), buf.Bytes())
}

func BenchmarkReadAllInitCap(b *testing.B) {
	buf := &bytes.Buffer{}
	for i := 0; i < 1024*1024; i++ {
		buf.Write([]byte("hello world!"))
	}
	result := buf.Bytes()
	size := buf.Len()
	for i := 0; i < b.N; i++ {
		data, err := ReadAllInitCap(bytes.NewBuffer(result), 1024*1024)
		if err != nil {
			panic(err)
		}
		if len(data) != size {
			panic(errors.New("data is invalid"))
		}
	}
}
