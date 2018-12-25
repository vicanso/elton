package middleware

import (
	"github.com/vicanso/cod"
)

type (
	// Skipper check for skip middleware
	Skipper func(c *cod.Context) bool
)

// DefaultSkipper default skiper function(not skip)
func DefaultSkipper(c *cod.Context) bool {
	return false
}
