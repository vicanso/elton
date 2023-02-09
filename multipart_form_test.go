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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultipartForm(t *testing.T) {
	assert := assert.New(t)

	form := NewMultipartForm()
	defer func() {
		_ = form.Destroy()
	}()
	err := form.AddField("a", "b")
	assert.Nil(err)

	err = form.AddFile("file", "test.txt", bytes.NewBufferString("Hello world!"))
	assert.Nil(err)

	r, err := form.Reader()
	assert.Nil(err)
	buf, err := io.ReadAll(r)
	assert.Nil(err)
	str := string(buf)
	assert.True(strings.Contains(str, "Hello world!"))
	assert.True(strings.Contains(str, `Content-Disposition: form-data; name="file"; filename="test.txt"`))
	assert.True(strings.Contains(str, `Content-Type: application/octet-stream`))
	assert.True(strings.Contains(str, `Content-Disposition: form-data; name="a"`))
}
