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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleSignedKeys(t *testing.T) {
	assert := assert.New(t)
	var sk SignedKeysGenerator
	keys := []string{
		"a",
		"b",
	}
	sk = new(SimpleSignedKeys)
	sk.SetKeys(keys)
	assert.Equal(sk.GetKeys(), keys)
}

func TestRWMutexSignedKeys(t *testing.T) {
	assert := assert.New(t)
	var sk SignedKeysGenerator
	keys := []string{
		"a",
		"b",
	}
	sk = new(RWMutexSignedKeys)
	sk.SetKeys(keys)
	assert.Equal(sk.GetKeys(), keys)
	done := make(chan bool)
	max := 10000
	go func() {
		for index := 0; index < max; index++ {
			sk.SetKeys([]string{"a"})
		}
		done <- true
	}()
	for index := 0; index < max; index++ {
		sk.GetKeys()
	}
	<-done
}

func TestAtomicSignedKeys(t *testing.T) {
	assert := assert.New(t)
	var sk SignedKeysGenerator
	keys := []string{
		"a",
		"b",
	}
	sk = new(AtomicSignedKeys)
	sk.SetKeys(keys)
	assert.Equal(sk.GetKeys(), keys)
	done := make(chan bool)
	max := 10000
	go func() {
		for index := 0; index < max; index++ {
			sk.SetKeys([]string{"a"})
		}
		done <- true
	}()
	for index := 0; index < max; index++ {
		keys := sk.GetKeys()
		if len(keys) == 2 {
			assert.Equal([]string{"a", "b"}, keys)
		} else {
			assert.Equal([]string{"a"}, keys)
		}
	}
	<-done
}

func BenchmarkRWMutexSignedKeys(b *testing.B) {
	sk := new(RWMutexSignedKeys)
	sk.SetKeys([]string{"a"})
	for i := 0; i < b.N; i++ {
		sk.GetKeys()
	}
}

func BenchmarkAtomicSignedKeys(b *testing.B) {
	sk := new(AtomicSignedKeys)
	sk.SetKeys([]string{"a"})
	for i := 0; i < b.N; i++ {
		sk.GetKeys()
	}
}
