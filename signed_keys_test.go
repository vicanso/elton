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
}
