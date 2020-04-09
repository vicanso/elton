package elton

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteParams(t *testing.T) {
	assert := assert.New(t)
	params := new(RouteParams)
	params.Add("id", "1")
	assert.Equal("1", params.Get("id"))
	assert.Equal(map[string]string{
		"id": "1",
	}, params.ToMap())
}
