// MIT License

// Copyright (c) 2023 Tree Xie

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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLruStore(t *testing.T) {
	assert := assert.New(t)

	store := NewLruStore(128)

	ctx := context.Background()
	key1 := "1"
	buf, _ := store.Get(ctx, key1)
	assert.Empty(buf)

	_ = store.Set(ctx, key1, []byte("Hello world!"), time.Second)
	buf, _ = store.Get(ctx, key1)
	assert.Equal([]byte("Hello world!"), buf)

	time.Sleep(2 * time.Second)
	buf, _ = store.Get(ctx, key1)
	assert.Empty(buf)
}

func BenchmarkLruStore(b *testing.B) {
	store := NewLruStore(128)
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		_ = store.Set(ctx, "key", []byte("Hello world!"), time.Second)
		_, _ = store.Get(ctx, "key")
	}
}
