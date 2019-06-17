package cod

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
		sk.GetKeys()
	}
	<-done
}

func BenchmarkRWMutexSignedKeys(b *testing.B) {
	var sk SignedKeysGenerator
	sk = new(RWMutexSignedKeys)
	sk.SetKeys([]string{"a"})
	for i := 0; i < b.N; i++ {
		sk.GetKeys()
	}
}

func BenchmarkAtomicSignedKeys(b *testing.B) {
	var sk SignedKeysGenerator
	sk = new(AtomicSignedKeys)
	sk.SetKeys([]string{"a"})
	for i := 0; i < b.N; i++ {
		sk.GetKeys()
	}
}
